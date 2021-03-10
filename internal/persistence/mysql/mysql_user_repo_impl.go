package mysql

import (
	"fmt"

	"github.com/yeqown/cassem/internal/persistence"

	"gorm.io/gorm"
)

var (
	_userTbl = new(persistence.UserDO)
)

type mysqlUserRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) persistence.UserRepository {
	return mysqlUserRepo{
		db: db,
	}
}

func (m mysqlUserRepo) Create(u *persistence.UserDO) error {
	return m.db.Model(_userTbl).
		Create(u).Error
}

func (m mysqlUserRepo) ResetPassword(account, passwordWithSalt string) error {
	return m.db.Model(_userTbl).
		Where("account = ?", account).
		Update("password", passwordWithSalt).Error
}

func (m mysqlUserRepo) QueryUser(account string) (*persistence.UserDO, error) {
	out := new(persistence.UserDO)
	err := m.db.Model(_userTbl).
		Where("account = ?", account).First(out).Error

	return out, err
}

func (m mysqlUserRepo) PagingUsers(filter *persistence.PagingUsersFilter) ([]*persistence.UserDO, int, error) {
	if filter == nil || filter.Limit <= 0 || filter.Offset < 0 {
		filter = &persistence.PagingUsersFilter{
			Limit:          10,
			Offset:         0,
			AccountPattern: "",
		}
	}

	userDOs := make([]*persistence.UserDO, 0, filter.Limit)
	tx := m.db.Model(_userTbl)
	if filter.AccountPattern != "" {
		tx = tx.Where("`account` LIKE ?", fmt.Sprintf("%%%s%%", filter.AccountPattern))
	}

	count := int64(0)
	err := tx.Order("created_at DESC").
		Count(&count).
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&userDOs).Error

	return userDOs, int(count), err
}

func (m mysqlUserRepo) Migrate() error {
	return m.db.AutoMigrate(
		&persistence.UserDO{},
	)
}
