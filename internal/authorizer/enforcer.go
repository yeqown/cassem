package authorizer

import (
	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/mysql"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var _MODEL = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && obj_match(r.obj, p.obj) && act_match(r.act, p.act)
`

type IEnforcer interface {
	Enforce(req *EnforceRequest) bool
	ListSubjectPolicies(subject string) []Policy
	UpdateSubjectPolicies(subject string, policies []Policy) error
}

type IAuthorizer interface {
	IEnforcer

	Migrate() error
}

type EnforceRequest struct {
	Subject string
	Object  string
	Action  string

	//Namespace string
	//Container string
	//Pair      string
}

// casbinAuthorities implement IAuthorizer based on casbin.ACL model.
type casbinAuthorities struct {
	aclEnforcer *casbin.Enforcer
	repo        persistence.Repository
}

func New(c *conf.MySQL) (auth IAuthorizer, err error) {
	repo, err := mysql.New(c)
	if err != nil {
		return nil, errors.Wrap(err, "authorizer.New could not load persistence")
	}
	a, err := repo.PolicyAdapter()
	if err != nil {
		return nil, err
	}

	// model
	m, err := model.NewModelFromString(_MODEL)
	if err != nil {
		return nil, err
	}

	// enforcer
	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		return nil, err
	}

	// options
	e.EnableAutoSave(true)
	if err = e.LoadPolicy(); err != nil {
		return nil, err
	}
	e.AddFunction("obj_match", func(args ...interface{}) (interface{}, error) {
		//log.
		//	WithFields(log.Fields{"args": args}).
		//	Debug("obj_match")

		rObj, pObj := args[0].(string), args[1].(string)
		if pObj == "any" {
			return true, nil
		}

		return rObj == pObj, nil
	})
	e.AddFunction("act_match", func(args ...interface{}) (interface{}, error) {
		//log.
		//	WithFields(log.Fields{"args": args}).
		//	Debug("act_match")

		rAct, pAct := args[0].(string), args[1].(string)
		if pAct == "any" {
			return true, nil
		}

		return rAct == pAct, nil
	})

	auth = casbinAuthorities{
		repo:        repo,
		aclEnforcer: e,
	}

	return
}

// Migrate ...
// DONE(@yeqown): migrate to init data, only be called cassemctl.
func (c casbinAuthorities) Migrate() error {
	// DONE(@yeqown) add root account automatically, and add all permissions to root account.
	u := &persistence.User{Account: "admin", PasswordWithSalt: "cassem", Name: "admin"}
	if err := c.repo.CreateUser(u); err != nil {
		return errors.Wrap(err, "failed to create root account")
	}

	token := NewToken(int(u.ID))
	if err := c.UpdateSubjectPolicies(token.Subject(), AllPolicies); err != nil {
		return errors.Wrap(err, "failed to assign all policy to root account")
	}

	return nil
}

func (c casbinAuthorities) Enforce(req *EnforceRequest) bool {
	log.
		WithField("req", req).
		Debug("casbinAuthorities.Enforce called")

	if req == nil {
		return false
	}

	allow, err := c.aclEnforcer.Enforce(req.Subject, req.Object, req.Action)
	if err != nil {
		log.
			WithFields(log.Fields{
				"subject": req.Subject,
				"object":  req.Object,
				"action":  req.Action,
			}).
			Errorf("casbinAuthorities.Enforce failed to enforce: %v", err)

		return allow
	}

	return allow
}

func (c casbinAuthorities) ListSubjectPolicies(subject string) []Policy {
	out := c.aclEnforcer.GetFilteredPolicy(0, subject)

	policies := make([]Policy, 0, len(out))
	for _, p := range out {
		// FIXED(@yeqown): guard invalid data source.
		if len(p) < 3 {
			log.
				WithFields(log.Fields{
					"policy": p,
				}).
				Warnf("casbinAuthorities.ListSubjectPolicies could not handle (length is less than 3)")
		}

		policies = append(policies, Policy{
			Subject: p[0],
			Object:  p[1],
			Action:  p[2],
		})
	}

	return policies
}

func (c casbinAuthorities) UpdateSubjectPolicies(subject string, policies []Policy) error {
	_, err := c.aclEnforcer.RemoveFilteredPolicy(0, subject)
	if err != nil {
		return err
	}

	in := make([][]string, 0, len(policies))
	for _, policy := range policies {
		if err = ValidPolicy(subject, policy); err != nil {
			log.WithFields(log.Fields{
				"subject": subject,
				"policy":  policy,
			}).Warnf("policy invalid, skip")
		}
		in = append(in, []string{policy.Subject, policy.Object, policy.Action})
	}

	_, err = c.aclEnforcer.AddPolicies(in)
	if err != nil {
		return err
	}

	return nil
}
