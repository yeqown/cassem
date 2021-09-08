package concept

import (
	"context"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	defaultrolemanager "github.com/casbin/casbin/v2/rbac/default-role-manager"
	"github.com/pkg/errors"
	"github.com/yeqown/log"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/hash"
)

const (
	Action_READ    = "r"
	Action_WRITE   = "w"
	Action_DELETE  = "d"
	Action_PUBLISH = "p"
	Action_ANY     = "*"
)

// domain is equal to app/env
const (
	Domain_ALL     = "*"
	Domain_CLUSTER = "cluster"
	// Domain_APP MUST NOT be used, this only represents the format of
	// app domain.
	Domain_APP = "app/env"
	// Domain_APP_ENV = "ae:appName/envName"
)

const (
	// Role_SUPERADMIN can control whole resources.
	// p superadmin * * *
	Role_SUPERADMIN = "superadmin"
	// Role_ADMIN is an admin role who owns all apps' all permissions.
	Role_ADMIN = "admin"
	// Role_APPOWNER can only control the app's resources which belong to him
	// and visit other apps's resources.
	Role_APPOWNER = "appowner"
	// Role_DEVELOPER can only access(except delete, publish, rollback permissions)
	// the app's resources which belong to him and visit other apps's resources.
	Role_DEVELOPER = "appdeveloper"
	// Role_VISITOR can only access(readonly) app's resources.
	Role_VISITOR = "visitor"
)

const (
	Object_USER    = "user"
	Object_ACL     = "acl"
	Object_APP     = "app"
	Object_ENV     = "env"
	Object_ELEMENT = "elem"
	Object_CLUSTER = "cluster"
	Object_ALL     = "*"
)

var (
	_ RBAC            = aclImpl{}
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
m = r.sub == 'superadmin' || \
	(g(r.sub, p.sub, r.dom) && \
	(p.dom == '*' || r.dom == p.dom) && \
	(p.obj == '*' || r.obj == p.obj) && \
	(p.act == '*' || r.act == p.act))
`

// g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act

type aclImpl struct {
	c apicassemdb.KVClient
	a *cassemAdapter
	e *casbin.Enforcer
}

// newRBAC construct a RBAC ACL interface.
func newRBAC(c apicassemdb.KVClient) (RBAC, error) {
	a := &cassemAdapter{cassemdb: c}

	m, err := model.NewModelFromString(_casbinModel)
	if err != nil {
		return nil, errors.Wrap(err, "concept.newRBAC.parseModel")
	}
	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		return nil, errors.Wrap(err, "concept.newRBAC.newEnforcer")
	}

	// use 1-layer RBAC
	e.SetRoleManager(defaultrolemanager.NewRoleManager(1))
	e.AddNamedDomainMatchingFunc("g", "", func(r, p string) bool {
		switch p {
		case Domain_ALL:
			return true
		case Domain_CLUSTER:
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

	return aclImpl{a: a, c: c, e: e}, nil
}

func (a aclImpl) GetUser(account string) (*User, error) {
	if strings.HasPrefix(account, "superadmin") {
		return &User{
			Account:        "superadmin",
			Nickname:       "superadmin",
			HashedPassword: "7c46f88749d0b4f39c0b089e67553361846cf9a0fa0213012ce345a5cfcea689",
			Salt:           "Y2Fzc2VuCg==",
			Status:         User_NORMAL,
		}, nil
	}

	r, err := a.c.GetKV(context.TODO(), &apicassemdb.GetKVReq{Key: genUserKey(account)})
	if err != nil {
		return nil, errors.Wrap(err, "aclImpl.GetUser")
	}

	u := new(User)
	apicassemdb.MustUnmarshal(r.GetEntity().GetVal(), u)

	roles, err := a.e.GetRolesForUser(account, Domain_ALL)
	log.
		WithFields(log.Fields{
			"roles": roles,
			"err":   err,
		}).
		Debugf("aclImpl.GetUser.GetRolesForUser")

	return u, nil
}

func (a aclImpl) AddUser(u *User) error {
	// encrypt user's password
	u.Salt = hash.RandKey(8)
	u.HashedPassword = hash.WithSalt(u.HashedPassword, u.Salt)

	// save
	data := apicassemdb.Must(apicassemdb.Marshal(u))
	r, err := a.c.SetKV(context.TODO(), &apicassemdb.SetKVReq{
		Key:       genUserKey(u.GetAccount()),
		IsDir:     false,
		Ttl:       apicassemdb.NEVER_EXPIRED,
		Val:       data,
		Overwrite: false,
	})
	if err != nil {
		return errors.Wrap(err, "aclImpl.AddUser")
	}
	_ = r

	return nil
}

func (a aclImpl) DisableUser(account string) error {
	r, err := a.c.GetKV(context.TODO(), &apicassemdb.GetKVReq{Key: genUserKey(account)})
	if err != nil {
		return errors.Wrap(err, "aclImpl.DisableUser")
	}

	u := new(User)
	apicassemdb.MustUnmarshal(r.GetEntity().GetVal(), u)
	u.Status = User_FORBIDDEN

	return a.saveUser(u)
}

func (a aclImpl) saveUser(u *User) error {
	data := apicassemdb.Must(apicassemdb.Marshal(u))
	r, err := a.c.SetKV(context.TODO(), &apicassemdb.SetKVReq{
		Key:       genUserKey(u.Account),
		IsDir:     false,
		Ttl:       apicassemdb.NEVER_EXPIRED,
		Val:       data,
		Overwrite: true,
	})
	if err != nil {
		return errors.Wrap(err, "aclImpl.saveUser")
	}
	_ = r

	return nil
}

func (a aclImpl) AssignRole(account, role string, domain ...string) error {
	assigned, err := a.e.AddRoleForUser(account, role, domain...)
	if err != nil {
		return errors.Wrap(err, "aclImpl.AssignRole")
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
	assigned, err := a.e.DeleteRoleForUser(account, role, domain...)
	if err != nil {
		return errors.Wrap(err, "aclImpl.RevokeRole")
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
	_, err := a.e.AddPolicies([][]string{
		{"superadmin", Domain_ALL, Object_ALL, Action_ANY},
		{"admin", Domain_ALL, Object_ALL, Action_READ},
		{"admin", Domain_ALL, Object_ALL, Action_WRITE},
	})
	return err
}

// cassemAdapter implements persist.Adapter of casbin acl model.
type cassemAdapter struct {
	cassemdb apicassemdb.KVClient
}

func (c cassemAdapter) LoadPolicy(model model.Model) error {
	r, err := c.cassemdb.GetKV(
		context.TODO(),
		&apicassemdb.GetKVReq{Key: genAclPolicyKey()},
	)
	if err != nil {
		if errors.Is(err, errorx.Err_NOT_FOUND) {
			return nil
		}

		return errors.Wrap(err, "cassemAdapter.LoadPolicy")
	}

	// c.casbinEntity = r.GetEntity()
	s := new(Casbin)
	apicassemdb.MustUnmarshal(r.GetEntity().GetVal(), s)

	for _, p := range s.GetPolicies() {
		loadPolicyLine(p, model)
	}

	return nil
}

func loadPolicyLine(policy *Casbin_Policy, model model.Model) {
	lineText := policy.Ptype
	if policy.V0 != "" {
		lineText += ", " + policy.V0
	}
	if policy.V1 != "" {
		lineText += ", " + policy.V1
	}
	if policy.V2 != "" {
		lineText += ", " + policy.V2
	}
	if policy.V3 != "" {
		lineText += ", " + policy.V3
	}
	if policy.V4 != "" {
		lineText += ", " + policy.V4
	}
	if policy.V5 != "" {
		lineText += ", " + policy.V5
	}

	persist.LoadPolicyLine(lineText, model)
}

func (c cassemAdapter) SavePolicy(model model.Model) error {
	s := &Casbin{
		Policies: make([]*Casbin_Policy, 0, len(model["p"])+len(model["g"])),
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
		Key:       genAclPolicyKey(),
		IsDir:     false,
		Ttl:       0,
		Val:       data,
		Overwrite: true,
	})
	if err != nil {
		return errors.Wrap(err, "cassemAdapter.SavePolicy")
	}

	return nil
}

func savePolicyLine(ptype string, rule []string) *Casbin_Policy {
	line := new(Casbin_Policy)

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
