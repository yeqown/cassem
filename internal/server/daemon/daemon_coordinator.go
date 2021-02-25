package daemon

import (
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
	coord "github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/pkg/datatypes"
	"github.com/yeqown/log"
)

//type Daemon struct {
//	repo persistence.Repository
//}
//
//func New(repo persistence.Repository) ICoordinator {
//	return Daemon{
//		repo: repo,
//	}
//}

func (d Daemon) GetContainer(key, ns string) (datatypes.IContainer, error) {
	v, err := d.repo.GetContainer(ns, key)
	if err != nil {
		return nil, errors.Wrap(err, "Daemon.GetContainer failed to get container")
	}

	return d.repo.Converter().ToContainer(v)
}

func (d Daemon) PagingContainers(filter *coord.FilterContainersOption) ([]datatypes.IContainer, int, error) {
	outs, count, err := d.repo.PagingContainers(&persistence.PagingContainersFilter{
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		KeyPattern: filter.KeyPattern,
		Namespace:  filter.Namespace,
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "Daemon.PagingContainers failed to paging pairs")
	}

	containers := make([]datatypes.IContainer, 0, len(outs))
	for _, v := range outs {
		p, err := d.repo.Converter().ToContainer(v)
		if err != nil {
			log.
				WithFields(log.Fields{
					"containerSource": v,
				}).
				Warnf("Daemon.PagingContainers could not convert pair: %v", err)
			continue
		}
		containers = append(containers, p)
	}

	return containers, count, nil
}

func (d Daemon) SaveContainer(container datatypes.IContainer) error {
	v, err := d.repo.Converter().FromContainer(container)
	if err != nil {
		return errors.Wrap(err, "Daemon.SaveContainer failed to convert container")
	}

	return d.repo.SaveContainer(v, true)
}

func (d Daemon) RemoveContainer(key string, ns string) error {
	return d.repo.RemoveContainer(ns, key)
}

// PagingNamespaces list namespaces those conform to the filter(FilterNamespacesOption).
//
// DONE(@yeqown): handle count return value
func (d Daemon) PagingNamespaces(filter *coord.FilterNamespacesOption) ([]string, int, error) {
	ns, count, err := d.repo.PagingNamespace(&persistence.PagingNamespacesFilter{
		Limit:            filter.Limit,
		Offset:           filter.Offset,
		NamespacePattern: filter.NamespacePattern,
	})

	return ns, count, err
}

func (d Daemon) SaveNamespace(ns string) error {
	return d.repo.SaveNamespace(ns)
}

func (d Daemon) GetPair(key, ns string) (datatypes.IPair, error) {
	v, err := d.repo.GetPair(ns, key)
	if err != nil {
		return nil, errors.Wrap(err, "Daemon.GetPair failed to get pair")
	}

	return d.repo.Converter().ToPair(v)
}

func (d Daemon) PagingPairs(filter *coord.FilterPairsOption) ([]datatypes.IPair, int, error) {
	outs, count, err := d.repo.PagingPairs(&persistence.PagingPairsFilter{
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		KeyPattern: filter.KeyPattern,
		Namespace:  filter.Namespace,
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "Daemon.PagingPairs failed to paging pairs")
	}

	pairs := make([]datatypes.IPair, 0, len(outs))
	for _, v := range outs {
		p, err := d.repo.Converter().ToPair(v)
		if err != nil {
			log.
				WithFields(log.Fields{
					"pairSource": v,
				}).
				Warnf("Daemon.PagingPairs could not convert pair: %v", err)
			continue
		}
		pairs = append(pairs, p)
	}

	return pairs, count, nil
}

func (d Daemon) SavePair(p datatypes.IPair) error {
	v, err := d.repo.Converter().FromPair(p)
	if err != nil {
		return errors.Wrap(err, "Daemon.SavePair failed to convert pair")
	}

	return d.repo.SavePair(v, true)
}

func (d Daemon) AddNode(serverId, addr string) error {
	log.Infof("received tryJoinCluster request for remote node %s, addr %s", serverId, addr)

	cf := d.raft.GetConfiguration()
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

	f := d.raft.AddVoter(raft.ServerID(serverId), raft.ServerAddress(addr), 0, 0)
	if err := f.Error(); err != nil {
		return err
	}

	log.Infof("node %s at %s joinedCluster successfully", serverId, addr)
	return nil
}

func (d Daemon) RemoveNode(nodeID string) error {
	log.Infof("received tryJoinCluster request for remote node %s", nodeID)

	cf := d.raft.GetConfiguration()
	if err := cf.Error(); err != nil {
		log.Errorf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range cf.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) {
			f := d.raft.RemoveServer(srv.ID, 0, 0)
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
