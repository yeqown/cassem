package authorizer

import (
	"strconv"

	"github.com/yeqown/cassem/internal/persistence"

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
	IAuthorizeManager

	Enforce(req *EnforceRequest) bool
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

// Token is the bridge between authorizer and HTTP API.
type Token struct {
	UserId int
}

func (t Token) Subject() string {
	return "uid:" + strconv.Itoa(t.UserId)
}

// IAuthorizeManager manages user, roles.
type IAuthorizeManager interface {
	Migrate() error

	// user permissions manage API
	ListSubjectPolicies(subject string) []Policy
	UpdateSubjectPolicies(subject string, policies []Policy) error

	// user and session manage API
	AddUser(account, password, name string) error
	Login(account, password string) (*persistence.UserDO, string, error)
	Session(tokenString string) (*Token, error)
	ResetPassword(account, password string) error
	PagingUsers(limit, offset int, accountPattern string) ([]*persistence.UserDO, int, error)
}
