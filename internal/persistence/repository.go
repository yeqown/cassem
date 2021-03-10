package persistence

import (
	"github.com/yeqown/cassem/pkg/datatypes"
	"gorm.io/gorm"
)

// Repository is a proxy who helps convert data between logic and persistence.Not only all parameters of Repository
// are logic datatype, but also all return values.
// NOTE(@yeqown): how to delete resource or mark it as deprecated, now only support container deletion.
type Repository interface {
	// datatypes.IContainer includes container properties: key, ns, fields
	GetContainer(ns, containerKey string) (interface{}, error)
	SaveContainer(c interface{}, update bool) error
	PagingContainers(filter *PagingContainersFilter) ([]interface{}, int, error)
	RemoveContainer(ns, containerKey string) error // DONE(@yeqown): container could be deleted
	UpdateContainerCheckSum(ns, key, checksum string) error

	// datatypes.IPair includes key-value pair data.
	GetPair(ns, key string) (interface{}, error)
	SavePair(v interface{}, update bool) error
	PagingPairs(filter *PagingPairsFilter) ([]interface{}, int, error)

	// namespace is a string represent the unique data domain of each data in cassem.
	PagingNamespace(filter *PagingNamespacesFilter) ([]string, int, error)
	SaveNamespace(ns string) error

	// Converter
	Converter() Converter

	Migrate() error
}

// DONE(@yeqown) design Converter to unbind Repository and logic datatype in repository's logic.
// Converter's purpose is to abstract conversion between repository and logic datatype.
type Converter interface {
	FromPair(p datatypes.IPair) (interface{}, error)
	ToPair(v interface{}) (datatypes.IPair, error)

	FromContainer(c datatypes.IContainer) (interface{}, error)
	ToContainer(v interface{}) (datatypes.IContainer, error)
}

type PagingContainersFilter struct {
	Limit  int
	Offset int
	// Namespace to filter pairs of current Namespace, DO NOT using fuzzy comparison.
	Namespace  string
	KeyPattern string
}

type PagingPairsFilter struct {
	Limit      int
	Offset     int
	KeyPattern string
	// Namespace to filter pairs of current Namespace, DO NOT using fuzzy comparison.
	Namespace string
}

type PagingNamespacesFilter struct {
	Limit            int
	Offset           int
	NamespacePattern string
}

type UserRepository interface {
	Create(u *UserDO) error

	ResetPassword(account, passwordWithSalt string) error

	QueryUser(account string) (*UserDO, error)

	PagingUsers(filter *PagingUsersFilter) ([]*UserDO, int, error)

	Migrate() error
}

type PagingUsersFilter struct {
	Limit          int
	Offset         int
	AccountPattern string
}

type UserDO struct {
	gorm.Model

	Account          string `gorm:"column:account;type:varchar(32);uniqueIndex:idx_unique_account"`
	PasswordWithSalt string `gorm:"column:password;type:varchar(64)"`
	Name             string `gorm:"column:name;type:varchar(16)"`
}

func (m UserDO) TableName() string {
	return "cassem_user"
}
