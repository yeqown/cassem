package authorizer

import (
	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/mysql"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
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

// casbinAuthorities implement IAuthorizer based on casbin.ACL model.
type casbinAuthorities struct {
	aclEnforcer *casbin.Enforcer
	userRepo    persistence.UserRepository
}

func New(c *conf.MySQL) (auth IAuthorizer, err error) {
	db, err := mysql.Connect(c)
	if err != nil {
		return nil, errors.Wrap(err, "authorizer.New could not connect to DB")
	}

	// adapter
	a, err := gormadapter.NewAdapterByDBUseTableName(db, "cassem", "permission_policy")
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
		userRepo:    mysql.NewUserRepository(db),
		aclEnforcer: e,
	}

	return
}

// Migrate ...
// DONE(@yeqown): migrate to init data, only be called cassemctl.
func (c casbinAuthorities) Migrate() error {
	if err := c.userRepo.Migrate(); err != nil {
		return errors.Wrap(err, "failed to migrate user table")
	}

	// TODO(@yeqown) add root account automatically, and add all permissions to root account.
	//if err := c.aclEnforcer.Migrate(); err != nil {
	//	return errors.Wrap(err, "failed to migrate ACL tables")
	//}

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

func (c casbinAuthorities) UpdateSubjectPolicies(subject string, policies []Policy) error {
	_, err := c.aclEnforcer.RemoveFilteredPolicy(0, subject)
	if err != nil {
		return err
	}

	in := make([][]string, 0, len(policies))
	for _, policy := range policies {
		if err = validPolicy(subject, policy); err != nil {
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

func (c casbinAuthorities) ListSubjectPolicies(subject string) []Policy {
	out := c.aclEnforcer.GetFilteredPolicy(0, subject)

	policies := make([]Policy, 0, len(out))
	for _, v := range out {
		// FIXME(@yeqown): guard invalid data source.
		policies = append(policies, Policy{
			Subject: v[0],
			Object:  v[1],
			Action:  v[2],
		})
	}

	return policies
}