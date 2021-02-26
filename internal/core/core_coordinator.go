package core

import (
	"bytes"
	"encoding/json"

	"github.com/yeqown/cassem/internal/cache"

	coord "github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/hashicorp/raft"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var (
	_ coord.ICoordinator = Core{}

	ErrNotLeader = errors.New("current node is not allow to write, " +
		"TODO(@yeqown) server proxy request to server")
)

// TODO(@yeqown): query from cache first and also sync to other node if missed.
func (c Core) GetContainer(key, ns string) (datatypes.IContainer, error) {
	//hit, err := c.cache.Query(key, ns)

	v, err := c.repo.GetContainer(ns, key)
	if err != nil {
		return nil, errors.Wrap(err, "Core.GetContainer failed to get container")
	}

	return c.repo.Converter().ToContainer(v)
}

// DownloadContainer query formatted container data at first, if not hit or got unexpected error, normal process
// would be used, if all process works well, core set into cache. plz notice that, the cache is distributed based on
// RAFT's FSM.
func (c Core) DownloadContainer(key, ns, format string) ([]byte, error) {
	cacheKey := key + "#" + ns + "#" + format
	data, err := c.containerCache.Get(cacheKey)
	switch err {
	case nil:
		log.
			WithField("cacheKey", cacheKey).
			Debug("container cache hit")
		return data, nil
	default:
		log.
			WithField("cacheKey", cacheKey).
			Debug("get cache failed")
	case cache.ErrMiss:
		log.
			WithField("cacheKey", cacheKey).
			Debug("container cache missed")
	}

	v, err := c.GetContainer(key, ns)
	if err != nil {
		return nil, err
	}

	container, err := c.repo.Converter().ToContainer(v)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	switch format {
	case "json":
		encoder := json.NewEncoder(buf)
		encoder.SetIndent("", "\t")
		err = encoder.Encode(container.ToMarshalInterface())
	case "toml":
		err = toml.
			NewEncoder(buf).
			Indentation("\t").
			Encode(container.ToMarshalInterface())
	default:
		err = errors.New("unsupported file format")
	}
	if err != nil {
		return nil, err
	}

	data = buf.Bytes()
	evicted, err := c.containerCache.Set(cacheKey, data)
	if err != nil {
		log.
			WithField("cacheKey", cacheKey).
			Error("could not set container cache")
	}
	if evicted {
		// TODO(@yeqown): should call raft to synchronous other nodes' data. apply from here.
		// means cache replacing happened
		// f := s.raft.Apply(msg, 10*time.Second)
	}

	return data, nil
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
	if !c.couldWrite() {
		// TODO(@yeqown): forwarding request to leader server
		return ErrNotLeader
	}

	v, err := c.repo.Converter().FromContainer(container)
	if err != nil {
		return errors.Wrap(err, "Core.SaveContainer failed to convert container")
	}

	return c.repo.SaveContainer(v, true)
}

func (c Core) RemoveContainer(key string, ns string) error {
	if !c.couldWrite() {
		// TODO(@yeqown): forwarding request to leader server
		return ErrNotLeader
	}

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
	if !c.couldWrite() {
		// TODO(@yeqown): forwarding request to leader server
		return ErrNotLeader
	}

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
	if !c.couldWrite() {
		// TODO(@yeqown): forwarding request to leader server
		return ErrNotLeader
	}

	v, err := c.repo.Converter().FromPair(p)
	if err != nil {
		return errors.Wrap(err, "Core.SavePair failed to convert pair")
	}

	return c.repo.SavePair(v, true)
}

func (c Core) AddNode(serverId, addr string) error {
	log.Infof("received AddNode request for remote node %s, addr %s", serverId, addr)

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
	log.Infof("received RemoveNode request for remote node %s", nodeID)

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

// TODO(@yeqown): let node be notified while leader changes, and also mark current node is leader or not?
func (c Core) watchLeaderChanges() {
}

// couldWrite only return true if current node is leader.
func (c Core) couldWrite() bool {
	return c.raft.State() == raft.Leader
}
