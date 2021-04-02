package bbolt

import (
	"github.com/yeqown/cassem/internal/persistence"

	"github.com/casbin/casbin/v2/persist"
)

func (b bboltRepoImpl) CreateUser(u *persistence.User) error {
	panic("implement me")
}

func (b bboltRepoImpl) ResetPassword(account, passwordWithSalt string) error {
	panic("implement me")
}

func (b bboltRepoImpl) QueryUser(account string) (*persistence.User, error) {
	panic("implement me")
}

func (b bboltRepoImpl) PagingUsers(filter *persistence.PagingUsersFilter) ([]*persistence.User, int, error) {
	panic("implement me")
}

func (b bboltRepoImpl) PolicyAdapter() (persist.Adapter, error) {
	panic("implement me")
}
