package authorizer

import (
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/pkg/hash"

	"github.com/pkg/errors"
)

var (
	_ IUserAndPolicyManager = casbinAuthorities{}
)

// IContainerAndPairManager provides the ability to manage users and policies, basically, it allow you to
// create, update, read and delete those resources.
type IUserAndPolicyManager interface {
	// user permissions manage API
	ListSubjectPolicies(subject string) []Policy
	UpdateSubjectPolicies(subject string, policies []Policy) error

	// user and session manage API
	AddUser(account, password, name string) (*persistence.User, error)
	Login(account, password string) (*persistence.User, string, error)
	ResetPassword(account, password string) error
	PagingUsers(limit, offset int, accountPattern string) ([]*persistence.User, int, error)
}

func (c casbinAuthorities) AddUser(account, password, name string) (*persistence.User, error) {
	u := &persistence.User{
		Account:          account,
		PasswordWithSalt: hash.WithSalt(password, "cassem"),
		Name:             name,
	}

	return u, c.repo.CreateUser(u)
}

func (c casbinAuthorities) Login(account, password string) (*persistence.User, string, error) {
	u, err := c.repo.QueryUser(account)
	if err != nil {
		return nil, "", err
	}

	if u.PasswordWithSalt != hash.WithSalt(password, "cassem") {
		return nil, "", errors.New("account and password could not match")
	}

	// DONE(@yeqown): generate jwt token
	token, err := genToken(u)
	return u, token, err
}

func (c casbinAuthorities) ResetPassword(account, password string) error {
	return c.repo.ResetPassword(account, hash.WithSalt(password, "cassem"))
}

func (c casbinAuthorities) PagingUsers(limit, offset int, accountPattern string) ([]*persistence.User, int, error) {
	return c.repo.PagingUsers(&persistence.PagingUsersFilter{
		Limit:          limit,
		Offset:         offset,
		AccountPattern: accountPattern,
	})
}
