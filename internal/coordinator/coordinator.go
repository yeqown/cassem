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
	DownloadContainer(key, ns string, format datatypes.ContainerFormat) ([]byte, error)
	PagingContainers(filter *FilterContainersOption) ([]datatypes.IContainer, int, error)
	SaveContainer(c datatypes.IContainer) error
	RemoveContainer(key string, ns string) error

	PagingNamespaces(filter *FilterNamespacesOption) ([]string, int, error)
	SaveNamespace(ns string) error

	GetPair(key, ns string) (datatypes.IPair, error)
	PagingPairs(filter *FilterPairsOption) ([]datatypes.IPair, int, error)
	SavePair(p datatypes.IPair) error
}

// IRaftCluster restrict what methods should coordinator should provide to API, so that API layer can
// provide internal API those helps cluster nodes to communicate with leader.
type IRaftCluster interface {
	AddNode(serverId, addr string) error

	RemoveNode(serverId string) error

	Apply(msg []byte) error

	// ShouldForwardToLeader returns shouldForward and current leaderAddr,
	// shouldForward = !(currentNode == Leader).
	ShouldForwardToLeader() (shouldForward bool, leaderAddr string)
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
