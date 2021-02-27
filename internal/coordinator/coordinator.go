package coord

import "github.com/yeqown/cassem/pkg/datatypes"

/*
 The coordinator's duty is to coordinate each component in cassem to work in flows. For example:
 ✈️ save-pair: web-server => coordinator.savePair => repository.savePair
                                                  => watcher.change (todo)
 And also control the behavior of each flow, On the opposite side, persistence.Repository should only be used to
 save and read metadata of each concept in cassem.
*/

// ICoordinator manage all flow from client to server.
type ICoordinator interface {
	IRaftCluster

	GetContainer(key, ns string) (datatypes.IContainer, error)
	DownloadContainer(key, ns, format string) ([]byte, error)
	PagingContainers(filter *FilterContainersOption) ([]datatypes.IContainer, int, error)
	SaveContainer(c datatypes.IContainer) error
	RemoveContainer(key string, ns string) error

	PagingNamespaces(filter *FilterNamespacesOption) ([]string, int, error)
	SaveNamespace(ns string) error

	GetPair(key, ns string) (datatypes.IPair, error)
	PagingPairs(filter *FilterPairsOption) ([]datatypes.IPair, int, error)
	SavePair(p datatypes.IPair) error
}

type IRaftCluster interface {
	AddNode(serverId, addr string) error

	RemoveNode(serverId string) error

	Apply(msg []byte) error
}

type FilterContainersOption struct {
	Limit      int
	Offset     int
	Namespace  string
	KeyPattern string
}

type FilterNamespacesOption struct {
	Limit            int
	Offset           int
	NamespacePattern string
}

type FilterPairsOption struct {
	Limit      int
	Offset     int
	KeyPattern string
	Namespace  string
}
