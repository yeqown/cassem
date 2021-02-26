package core

import (
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	coord "github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/pkg/datatypes"
	"github.com/yeqown/log"
)

//type Core struct {
//	repo persistence.Repository
//}
//
//func New(repo persistence.Repository) ICoordinator {
//	return Core{
//		repo: repo,
//	}
//}

func (c Core) GetContainer(key, ns string) (datatypes.IContainer, error) {
	v, err := c.repo.GetContainer(ns, key)
	if err != nil {
		return nil, errors.Wrap(err, "Core.GetContainer failed to get container")
	}

	return c.repo.Converter().ToContainer(v)
}

func (c Core) PagingContainers(filter *coord.FilterContainersOption) ([]datatypes.IContainer, int, error) {
	outs, count, err := c.repo.PagingContainers(&persistence.PagingContainersFilter{
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		KeyPattern: filter.KeyPattern,
		Namespace:  filter.Namespace,
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "Core.PagingContainers failed to paging pairs")
	}

	containers := make([]datatypes.IContainer, 0, len(outs))
	for _, v := range outs {
		p, err := c.repo.Converter().ToContainer(v)
		if err != nil {
			log.
				WithFields(log.Fields{
					"containerSource": v,
				}).
				Warnf("Core.PagingContainers could not convert pair: %v", err)
			continue
		}
		containers = append(containers, p)
	}

	return containers, count, nil
}

func (c Core) SaveContainer(container datatypes.IContainer) error {
	v, err := c.repo.Converter().FromContainer(container)
	if err != nil {
		return errors.Wrap(err, "Core.SaveContainer failed to convert container")
	}

	return c.repo.SaveContainer(v, true)
}

func (c Core) RemoveContainer(key string, ns string) error {
	return c.repo.RemoveContainer(ns, key)
}

// PagingNamespaces list namespaces those conform to the filter(FilterNamespacesOption).
//
// DONE(@yeqown): handle count return value
func (c Core) PagingNamespaces(filter *coord.FilterNamespacesOption) ([]string, int, error) {
	ns, count, err := c.repo.PagingNamespace(&persistence.PagingNamespacesFilter{
		Limit:            filter.Limit,
		Offset:           filter.Offset,
		NamespacePattern: filter.NamespacePattern,
	})

	return ns, count, err
}

func (c Core) SaveNamespace(ns string) error {
	return c.repo.SaveNamespace(ns)
}

func (c Core) GetPair(key, ns string) (datatypes.IPair, error) {
	v, err := c.repo.GetPair(ns, key)
	if err != nil {
		return nil, errors.Wrap(err, "Core.GetPair failed to get pair")
	}

	return c.repo.Converter().ToPair(v)
}

func (c Core) PagingPairs(filter *coord.FilterPairsOption) ([]datatypes.IPair, int, error) {
	outs, count, err := c.repo.PagingPairs(&persistence.PagingPairsFilter{
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		KeyPattern: filter.KeyPattern,
		Namespace:  filter.Namespace,
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "Core.PagingPairs failed to paging pairs")
	}

	pairs := make([]datatypes.IPair, 0, len(outs))
	for _, v := range outs {
		p, err := c.repo.Converter().ToPair(v)
		if err != nil {
			log.
				WithFields(log.Fields{
					"pairSource": v,
				}).
				Warnf("Core.PagingPairs could not convert pair: %v", err)
			continue
		}
		pairs = append(pairs, p)
	}

	return pairs, count, nil
}

func (c Core) SavePair(p datatypes.IPair) error {
	v, err := c.repo.Converter().FromPair(p)
	if err != nil {
		return errors.Wrap(err, "Core.SavePair failed to convert pair")
	}

	return c.repo.SavePair(v, true)
}

func (c Core) AddNode(serverId, addr string) error {
	log.Infof("received tryJoinCluster request for remote node %s, addr %s", serverId, addr)

	cf := c.raft.GetConfiguration()
	if err := cf.Error(); err != nil {
		log.Errorf("failed to get raft configuration: %v", err)
		return err
	}

	for _, server := range cf.Configuration().Servers {
		if server.ID == raft.ServerID(serverId) {
			log.Infof("node %s already joinedCluster raft cluster", serverId)
			return nil
		}
	}

	f := c.raft.AddVoter(raft.ServerID(serverId), raft.ServerAddress(addr), 0, 0)
	if err := f.Error(); err != nil {
		return err
	}

	log.Infof("node %s at %s joinedCluster successfully", serverId, addr)
	return nil
}

func (c Core) RemoveNode(nodeID string) error {
	log.Infof("received tryJoinCluster request for remote node %s", nodeID)

	cf := c.raft.GetConfiguration()
	if err := cf.Error(); err != nil {
		log.Errorf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range cf.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) {
			f := c.raft.RemoveServer(srv.ID, 0, 0)
			if err := f.Error(); err != nil {
				log.Errorf("failed to remove srv %s, err: ", nodeID, err)
				return err
			}

			log.Infof("node %s left successfully", nodeID)
			return nil
		}
	}

	log.Infof("node %s not exists in raft group", nodeID)
	return nil
}
