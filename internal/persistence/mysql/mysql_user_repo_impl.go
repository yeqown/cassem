package mysql

import (
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
