package mysql

import (
	"fmt"

	"github.com/yeqown/cassem/internal/persistence"
)

var (
	_userTbl = new(persistence.User)
)

func toUserDO(user *persistence.User) *userDO {
	return &userDO{
		Account:          user.Account,
		PasswordWithSalt: user.PasswordWithSalt,
		Name:             user.Name,
	}
}

func fromUserDO(u *userDO) *persistence.User {
	return &persistence.User{
		CreatedAt:        u.CreatedAt,
		Account:          u.Account,
		PasswordWithSalt: u.PasswordWithSalt,
		Name:             u.Name,
	}
}

func (m mysqlRepo) CreateUser(u *persistence.User) error {
	do := toUserDO(u)

	return m.db.Model(_userTbl).
		Create(do).Error
}

func (m mysqlRepo) ResetPassword(account, passwordWithSalt string) error {
	return m.db.Model(_userTbl).
		Where("account = ?", account).
		Update("password", passwordWithSalt).Error
}

func (m mysqlRepo) QueryUser(account string) (*persistence.User, error) {
	out := new(userDO)
	err := m.db.Model(_userTbl).
		Where("account = ?", account).First(out).Error

	if err != nil {
		return nil, err
	}

	return fromUserDO(out), err
}

func (m mysqlRepo) PagingUsers(filter *persistence.PagingUsersFilter) ([]*persistence.User, int, error) {
	if filter == nil || filter.Limit <= 0 || filter.Offset < 0 {
		filter = &persistence.PagingUsersFilter{
			Limit:          10,
			Offset:         0,
			AccountPattern: "",
		}
	}

	dos := make([]*userDO, 0, filter.Limit)
	tx := m.db.Model(_userTbl)
	if filter.AccountPattern != "" {
		tx = tx.Where("`account` LIKE ?", fmt.Sprintf("%%%s%%", filter.AccountPattern))
	}

	count := int64(0)
	err := tx.Order("created_at DESC").
		Count(&count).
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&dos).Error

	if err != nil {
		return nil, 0, err
	}
	out := make([]*persistence.User, 0, len(dos))
	for _, v := range dos {
		out = append(out, fromUserDO(v))
	}

	return out, int(count), err
}
