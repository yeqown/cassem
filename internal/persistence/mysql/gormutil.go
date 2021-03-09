package mysql

import (
	"time"

	"github.com/yeqown/cassem/internal/conf"

	"github.com/pkg/errors"
	mysqld "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Connect(c *conf.MySQL) (*gorm.DB, error) {
	//cfg := gorm.Config{
	//	SkipDefaultTransaction:                   false,
	//	NamingStrategy:                           nil,
	//	FullSaveAssociations:                     false,
	//	Logger:                                   nil,
	//	NowFunc:                                  nil,
	//	DryRun:                                   false,
	//	PrepareStmt:                              false,
	//	DisableAutomaticPing:                     false,
	//	DisableForeignKeyConstraintWhenMigrating: false,
	//	DisableNestedTransaction:                 false,
	//	AllowGlobalUpdate:                        false,
	//	QueryFields:                              false,
	//	CreateBatchSize:                          0,
	//	ClauseBuilders:                           nil,
	//	ConnPool:                                 nil,
	//	Dialector:                                nil,
	//	Plugins:                                  nil,
	//}

	db, err := gorm.Open(mysqld.Open(c.DSN), nil)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "invalid sql.DB")
	}

	sqlDB.SetMaxIdleConns(c.MaxIdle)
	sqlDB.SetMaxOpenConns(c.MaxOpen)
	sqlDB.SetConnMaxLifetime(time.Duration(c.MaxLifeTime) * time.Second)

	if c.Debug {
		db = db.Debug()
	}

	return db, nil
}
