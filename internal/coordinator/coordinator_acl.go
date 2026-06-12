package coordinator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	defaultrolemanager "github.com/casbin/casbin/v2/rbac/default-role-manager"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/hash"
)

var (
	_ concept.RBAC    = aclImpl{}
	_ persist.Adapter = cassemAdapter{}
)

var _casbinModel = `
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && \
	(p.dom == '*' || r.dom == p.dom) && \
	(p.obj == '*' || r.obj == p.obj) && \
	(p.act == '*' || r.act == p.act)
`

// g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act

type aclImpl struct {
	c apicassemdb.KVClient
	a *cassemAdapter
	e *casbin.Enforcer
}

func rbacRole(role string) string {
	if strings.HasPrefix(role, "role:") {
		return role
	}
	return "role:" + role
}

// newRBAC construct a RBAC ACL interface.
func newRBAC(c apicassemdb.KVClient) (concept.RBAC, error) {
	a := &cassemAdapter{cassemdb: c}

	m, err := model.NewModelFromString(_casbinModel)
	if err != nil {
		return nil, fmt.Errorf("concept.newRBAC.parseModel: %w", err)
	}
	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		return nil, fmt.Errorf("concept.newRBAC.newEnforcer: %w", err)
	}

	// use 1-layer RBAC
	e.SetRoleManager(defaultrolemanager.NewRoleManager(1))
	e.AddNamedDomainMatchingFunc("g", "", func(r, p string) bool {
		switch p {
		case concept.Domain_ALL:
			return true
		case concept.Domain_CLUSTER:
			return r == p
		}

		// app/subdomain strategy
		parr := strings.Split(p, "/")
		rarr := strings.Split(r, "/")
		if len(parr) < 2 || len(rarr) < 2 {
			return false
		}

		pdomain, psub := parr[0], parr[1]
		rdomain, rsub := rarr[0], rarr[1]
		if psub == "*" {
			return pdomain == rdomain
		}

		return pdomain == rdomain && rsub == psub
	})
	e.EnableAutoBuildRoleLinks(true)
	e.EnableAutoSave(false) // TODO(@yeqown): support automatically save
	if err = e.LoadPolicy(); err != nil {
		return nil, err
	}

	acl := aclImpl{a: a, c: c, e: e}
	if err = acl.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("concept.newRBAC.autoMigrate: %w", err)
	}

	return acl, nil
}

func (a aclImpl) GetUser(account string) (*concept.User, error) {
	r, err := a.c.GetKV(context.TODO(), &apicassemdb.GetKVReq{Key: concept.GenUserKey(account)})
	if err != nil {
		return nil, fmt.Errorf("aclImpl.GetUser: %w", err)
	}

	u := new(concept.User)
	apicassemdb.MustUnmarshal(r.GetEntity().GetVal(), u)

	roles, err := a.e.GetRolesForUser(account, concept.Domain_ALL)
	log.
		WithFields(log.Fields{
			"roles": roles,
			"err":   err,
		}).
		Debugf("aclImpl.GetUser.GetRolesForUser")

	return u, nil
}

func (a aclImpl) AddUser(u *concept.User) error {
	// encrypt user's password
	u.Salt = hash.RandKey(8)
	u.HashedPassword = hash.WithSalt(u.HashedPassword, u.Salt)

	// save
	data := apicassemdb.Must(apicassemdb.Marshal(u))
	r, err := a.c.SetKV(context.TODO(), &apicassemdb.SetKVReq{
		Key:       concept.GenUserKey(u.GetAccount()),
		IsDir:     false,
		Ttl:       0,
		Val:       data,
		Overwrite: false,
	})
	if err != nil {
		return fmt.Errorf("aclImpl.AddUser: %w", err)
	}
	_ = r

	return nil
}

func (a aclImpl) DisableUser(account string) error {
	r, err := a.c.GetKV(context.TODO(), &apicassemdb.GetKVReq{Key: concept.GenUserKey(account)})
	if err != nil {
		return fmt.Errorf("aclImpl.DisableUser: %w", err)
	}

	u := new(concept.User)
	apicassemdb.MustUnmarshal(r.GetEntity().GetVal(), u)
	u.Status = concept.User_FORBIDDEN

	return a.saveUser(u)
}

func (a aclImpl) ResetUser(account, password string) error {
	if strings.HasPrefix(account, "superadmin") {
		return fmt.Errorf("could not reset superadmin: %w", errorx.Err_PERMISSION_DENIED)
	}

	r, err := a.c.GetKV(context.TODO(), &apicassemdb.GetKVReq{Key: concept.GenUserKey(account)})
	if err != nil {
		return fmt.Errorf("aclImpl.ResetUser: %w", err)
	}

	u := new(concept.User)
	apicassemdb.MustUnmarshal(r.GetEntity().GetVal(), u)
	u.Salt = hash.RandKey(8)
	u.HashedPassword = hash.WithSalt(password, u.Salt)
	u.Status = concept.User_NORMAL

	return a.saveUser(u)
}

func (a aclImpl) saveUser(u *concept.User) error {
	data := apicassemdb.Must(apicassemdb.Marshal(u))
	r, err := a.c.SetKV(context.TODO(), &apicassemdb.SetKVReq{
		Key:       concept.GenUserKey(u.Account),
		IsDir:     false,
		Ttl:       0,
		Val:       data,
		Overwrite: true,
	})
	if err != nil {
		return fmt.Errorf("aclImpl.saveUser: %w", err)
	}
	_ = r

	return nil
}

func (a aclImpl) AssignRole(account, role string, domain ...string) error {
	assigned, err := a.e.AddRoleForUser(account, rbacRole(role), domain...)
	if err != nil {
		return fmt.Errorf("aclImpl.AssignRole: %w", err)
	}

	if !assigned {
		return nil
	}

	if err = a.e.SavePolicy(); err != nil {
		log.
			WithFields(log.Fields{
				"account": account,
				"role":    role,
				"domain":  domain,
			}).
			Warn("aclImpl.AssignRole failed savePolicy")
	}

	return nil
}

func (a aclImpl) RevokeRole(account, role string, domain ...string) error {
	assigned, err := a.e.DeleteRoleForUser(account, rbacRole(role), domain...)
	if err != nil {
		return fmt.Errorf("aclImpl.RevokeRole: %w", err)
	}

	if !assigned {
		return nil
	}

	if err = a.e.SavePolicy(); err != nil {
		log.
			WithFields(log.Fields{
				"account": account,
				"role":    role,
				"domain":  domain,
			}).
			Warn("aclImpl.AssignRole failed savePolicy")
	}

	return nil
}

func (a aclImpl) Enforce(subject, domain, object, act string) (bool, error) {
	allow, err := a.e.Enforce(subject, domain, object, act)
	if err != nil {
		log.
			WithFields(log.Fields{
				"account": subject,
				"perm":    object,
				"act":     act,
				"error":   err,
			}).
			Errorf("aclImpl.Enforce failed")
	}

	log.
		WithFields(log.Fields{
			"account": subject,
			"domain":  domain,
			"perm":    object,
			"act":     act,
			"allow":   allow,
		}).
		Debug("aclImpl.Enforce called")

	return allow, nil
}

// AutoMigrate initialize builtin-role and permissions.
func (a aclImpl) AutoMigrate() error {
	changed := false
	for _, policy := range [][]string{
		{rbacRole(concept.Role_SUPERADMIN), concept.Domain_ALL, concept.Object_ALL, concept.Action_ANY},
		{rbacRole(concept.Role_ADMIN), concept.Domain_ALL, concept.Object_ALL, concept.Action_READ},
		{rbacRole(concept.Role_ADMIN), concept.Domain_ALL, concept.Object_ALL, concept.Action_WRITE},
	} {
		added, err := a.e.AddPolicy(policy)
		if err != nil {
			return err
		}
		changed = changed || added
	}
	if !changed {
		return nil
	}
	return a.e.SavePolicy()
}

func (a aclImpl) BootstrapAdmin(account, nickname, password string) error {
	if _, err := a.GetUser(account); err == nil {
		return a.AssignRole(account, concept.Role_SUPERADMIN, concept.Domain_ALL)
	} else if !errors.Is(err, errorx.Err_NOT_FOUND) {
		return fmt.Errorf("aclImpl.BootstrapAdmin: %w", err)
	}

	if err := a.AddUser(&concept.User{
		Account:        account,
		Nickname:       nickname,
		HashedPassword: password,
		Status:         concept.User_NORMAL,
	}); err != nil {
		return fmt.Errorf("aclImpl.BootstrapAdmin: %w", err)
	}
	return a.AssignRole(account, concept.Role_SUPERADMIN, concept.Domain_ALL)
}

// cassemAdapter implements persist.Adapter of casbin acl model.
type cassemAdapter struct {
	cassemdb apicassemdb.KVClient
}

func (c cassemAdapter) LoadPolicy(model model.Model) error {
	r, err := c.cassemdb.GetKV(
		context.TODO(),
		&apicassemdb.GetKVReq{Key: concept.GenAclPolicyKey()},
	)
	if err != nil {
		if errors.Is(err, errorx.Err_NOT_FOUND) {
			return nil
		}

		return fmt.Errorf("cassemAdapter.LoadPolicy: %w", err)
	}

	// c.casbinEntity = r.GetEntity()
	s := new(concept.Casbin)
	apicassemdb.MustUnmarshal(r.GetEntity().GetVal(), s)

	for _, p := range s.GetPolicies() {
		loadPolicyLine(p, model)
	}

	return nil
}

func loadPolicyLine(policy *concept.Casbin_Policy, model model.Model) {
	lineText := policy.Ptype
	values := []string{policy.V0, policy.V1, policy.V2, policy.V3, policy.V4, policy.V5}
	if policy.Ptype == "p" && values[0] != "" {
		values[0] = rbacRole(values[0])
	}
	if policy.Ptype == "g" && values[1] != "" {
		values[1] = rbacRole(values[1])
	}
	for _, value := range values {
		if value != "" {
			lineText += ", " + value
		}
	}

	if err := persist.LoadPolicyLine(lineText, model); err != nil {
		log.WithFields(log.Fields{"policy": lineText, "error": err}).Warn("cassemAdapter.loadPolicyLine failed")
	}
}

func (c cassemAdapter) SavePolicy(model model.Model) error {
	s := &concept.Casbin{
		Policies: make([]*concept.Casbin_Policy, 0, len(model["p"])+len(model["g"])),
	}

	for ptype, ast := range model["p"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			s.Policies = append(s.Policies, line)
		}
	}

	for ptype, ast := range model["g"] {
		for _, rule := range ast.Policy {
			line := savePolicyLine(ptype, rule)
			s.Policies = append(s.Policies, line)
		}
	}

	data := apicassemdb.Must(apicassemdb.Marshal(s))
	_, err := c.cassemdb.SetKV(context.TODO(), &apicassemdb.SetKVReq{
		Key:       concept.GenAclPolicyKey(),
		IsDir:     false,
		Ttl:       0,
		Val:       data,
		Overwrite: true,
	})
	if err != nil {
		return fmt.Errorf("cassemAdapter.SavePolicy: %w", err)
	}

	return nil
}

func savePolicyLine(ptype string, rule []string) *concept.Casbin_Policy {
	line := new(concept.Casbin_Policy)

	line.Ptype = ptype
	if len(rule) > 0 {
		line.V0 = rule[0]
	}
	if len(rule) > 1 {
		line.V1 = rule[1]
	}
	if len(rule) > 2 {
		line.V2 = rule[2]
	}
	if len(rule) > 3 {
		line.V3 = rule[3]
	}
	if len(rule) > 4 {
		line.V4 = rule[4]
	}
	if len(rule) > 5 {
		line.V5 = rule[5]
	}

	return line
}

func (c cassemAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	log.
		WithFields(log.Fields{
			"sec":   sec,
			"ptype": ptype,
			"rule":  rule,
		}).
		Debug("cassemAdapter.AddPolicy called")
	return errors.New("not implemented")
}

func (c cassemAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	log.
		WithFields(log.Fields{
			"sec":   sec,
			"ptype": ptype,
			"rule":  rule,
		}).
		Debug("cassemAdapter.RemovePolicy called")
	return errors.New("not implemented")
}

func (c cassemAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return errors.New("not implemented")
}
