package persistence

import (
	"time"

	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/casbin/casbin/v2/persist"
)

// Repository is a proxy who helps convert data between logic and persistence.Not only all parameters of Repository
// are logic datatype, but also all return values.
// NOTE(@yeqown): how to delete resource or mark it as deprecated, now only support container deletion.
type Repository interface {
	IContainerPairPersist
	IUserPersist
	IPolicyAdapter
	IMigrator
}

// DONE(@yeqown) design Converter to unbind Repository and logic datatype in repository's logic.
// Converter's purpose is to abstract conversion between repository and logic datatype.
//
// for example:
//
// 	v, err := Repository.GetContainer()
// 	v is a interface, only Converter knows how to parse it into datatypes.IPair
// 	p, err := Converter.ToPair(v)
//
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

type IContainerPairPersist interface {
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
}

type PagingUsersFilter struct {
	Limit          int
	Offset         int
	AccountPattern string
}

type User struct {
	ID               uint
	CreatedAt        time.Time
	Account          string
	PasswordWithSalt string
	Name             string
}

type IUserPersist interface {
	CreateUser(u *User) error
	ResetPassword(account, passwordWithSalt string) error
	QueryUser(account string) (*User, error)
	PagingUsers(filter *PagingUsersFilter) ([]*User, int, error)
}

type IPolicyAdapter interface {
	PolicyAdapter() (persist.Adapter, error)
}

type IMigrator interface {
	Migrate() error
}
