package authorizer

import (
	"github.com/pkg/errors"
)

const (
	// Actions
	ACTION_READ  = "read"
	ACTION_WRITE = "write"
	ACTION_ANY   = "any"

	// Objects
	OBJ_NAMESPACE = "namespace"
	OBJ_CONTAINER = "container"
	OBJ_PAIR      = "pair"
	OBJ_USER      = "user"
	OBJ_POLICY    = "policy"
	OBJ_ANY       = "any"
)

var (
	allPolicies = []Policy{
		{Subject: "", Object: OBJ_NAMESPACE, Action: ACTION_ANY},
		{Subject: "", Object: OBJ_CONTAINER, Action: ACTION_ANY},
		{Subject: "", Object: OBJ_PAIR, Action: ACTION_ANY},
		{Subject: "", Object: OBJ_USER, Action: ACTION_ANY},
		{Subject: "", Object: OBJ_POLICY, Action: ACTION_ANY},
	}

	defaultPolicies = []Policy{
		{Subject: "", Object: OBJ_NAMESPACE, Action: ACTION_READ},
		{Subject: "", Object: OBJ_CONTAINER, Action: ACTION_READ},
		{Subject: "", Object: OBJ_PAIR, Action: ACTION_READ},
	}
)

type EnforceRequest struct {
	Subject string
	Object  string
	Action  string

	//Namespace string
	//Container string
	//Pair      string
}

// IAuthorizer
type IAuthorizer interface {
	IUserAndPolicyManager
	IEnforcer

	Migrate() error
}

// Policy contains whole data what describes a ACL rule.
type Policy struct {
	Subject string
	Object  string
	Action  string
}

func validPolicy(subject string, p Policy) (err error) {
	var errmsg string
	defer func() {
		if errmsg != "" {
			err = errors.New(errmsg)
			return
		}
	}()

	if p.Subject != subject {
		errmsg = "inconsistent subject"
		return
	}

	if p.Object == "" || p.Subject == "" || p.Action == "" {
		errmsg = "incomplete policy"
		return
	}

	return
}
