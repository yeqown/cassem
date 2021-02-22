package persistence

import "github.com/yeqown/cassem/pkg/datatypes"

// Repository is a proxy who helps convert data between logic and persistence.Not only all parameters of Repository
// are logic datatype, but also all return values.
// TODO(@yeqown): how to delete resource or mark it as deprecated ?
type Repository interface {
	// datatypes.IContainer includes container properties: key, ns, fields
	GetContainer(ns, containerKey string) (interface{}, error)
	SaveContainer(c interface{}, isUpdate bool) error
	PagingContainers(filter *PagingContainersFilter) ([]interface{}, int, error)

	// datatypes.IPair includes key-value pair data.
	GetPair(ns, key string) (interface{}, error)
	SavePair(v interface{}, isUpdate bool) error
	PagingPairs(filter *PagingPairsFilter) ([]interface{}, int, error)

	// namespace is a string represent the unique data domain of each data in cassem.
	PagingNamespace(filter *PagingNamespacesFilter) ([]string, error)
	SaveNamespace(ns string) error

	// Converter
	Converter() Converter
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
	Limit      int
	Offset     int
	Namespace  string
	KeyPattern string
}

type PagingPairsFilter struct {
	Limit      int
	Offset     int
	KeyPattern string
	Namespace  string
}

type PagingNamespacesFilter struct {
	Limit            int
	Offset           int
	NamespacePattern string
}
