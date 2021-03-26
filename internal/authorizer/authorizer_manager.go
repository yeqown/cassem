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
	AddUser(account, password, name string) (*persistence.UserDO, error)
	Login(account, password string) (*persistence.UserDO, string, error)
	ResetPassword(account, password string) error
	PagingUsers(limit, offset int, accountPattern string) ([]*persistence.UserDO, int, error)
}

func (c casbinAuthorities) AddUser(account, password, name string) (*persistence.UserDO, error) {
	u := &persistence.UserDO{
		Account:          account,
		PasswordWithSalt: hash.WithSalt(password, "cassem"),
		Name:             name,
	}

	return u, c.userRepo.Create(u)
}

func (c casbinAuthorities) Login(account, password string) (*persistence.UserDO, string, error) {
	u, err := c.userRepo.QueryUser(account)
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
	return c.userRepo.ResetPassword(account, hash.WithSalt(password, "cassem"))
}

func (c casbinAuthorities) PagingUsers(limit, offset int, accountPattern string) ([]*persistence.UserDO, int, error) {
	return c.userRepo.PagingUsers(&persistence.PagingUsersFilter{
		Limit:          limit,
		Offset:         offset,
		AccountPattern: accountPattern,
	})
}
