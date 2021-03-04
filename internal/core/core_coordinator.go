package core

import (
	"bytes"
	"encoding/json"
	"time"

	coord "github.com/yeqown/cassem/internal/coordinator"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/watcher"
	"github.com/yeqown/cassem/pkg/datatypes"
	"github.com/yeqown/cassem/pkg/hash"

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
func (c Core) DownloadContainer(key, ns string, format datatypes.ContainerFormat) ([]byte, error) {
	cacheKey := c.genContainerCacheKey(ns, key, format)
	hit, data := c.getContainerCache(cacheKey)
	if hit {
		return data, nil
	}

	container, err := c.GetContainer(key, ns)
	if err != nil {
		return nil, errors.Wrap(err, "Core.DownloadContainer failed to get container")
	}

	buf := bytes.NewBuffer(nil)
	switch format {
	case datatypes.JSON:
		encoder := json.NewEncoder(buf)
		encoder.SetIndent("", "\t")
		err = encoder.Encode(container.ToMarshalInterface())
	case datatypes.TOML:
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
	// OPTIMISZE(@yeqown) set cache asynchronously, so download response quickly.
	go c.setContainerCache(cacheKey, data)

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

// SaveContainer save container into persistence, but at the same time would trigger watcher if there is any
// changes of current container. Notice that only SaveContainer would trigger IWatcher.ChangesNotify with thought what
// pairs would be changed with low frequency.
func (c Core) SaveContainer(container datatypes.IContainer) error {
	if !c.isLeader() {
		return ErrNotLeader
	}

	v, err := c.repo.Converter().FromContainer(container)
	if err != nil {
		return errors.Wrap(err, "Core.SaveContainer failed to convert container")
	}

	err = c.repo.SaveContainer(v, true)

	// Core.watchContainerChanges would check and update container.checksum
	// if checksum changed means container changes happened.
	go startWithRecover("watchContainerChanges", func() error {
		return c.watchContainerChanges(container.NS(), container.Key())
	})

	return err
}

func (c Core) RemoveContainer(key string, ns string) error {
	if !c.isLeader() {
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
	if !c.isLeader() {
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
	if !c.isLeader() {
		return ErrNotLeader
	}

	v, err := c.repo.Converter().FromPair(p)
	if err != nil {
		return errors.Wrap(err, "Core.SavePair failed to convert pair")
	}

	return c.repo.SavePair(v, true)
}

// AddNode only leader node would receive such request. MAYBE?
func (c Core) AddNode(serverId, addr string) error {
	log.Infof("received AddNode request for remote node %s, addr %s", serverId, addr)

	if !c.isLeader() {
		log.
			Warn("RemoveNode request should not be executed by nonleader node")

		return ErrNotLeader
	}

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

// RemoveNode only leader node would receive such request.
func (c Core) RemoveNode(nodeID string) error {
	log.Infof("received RemoveNode request for remote node %s", nodeID)

	if !c.isLeader() {
		log.
			Warn("RemoveNode request should not be executed by nonleader node")

		return ErrNotLeader
	}

	cf := c.raft.GetConfiguration()
	if err := cf.Error(); err != nil {
		log.Errorf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range cf.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) {
			f := c.raft.RemoveServer(srv.ID, 0, 0)
			if err := f.Error(); err != nil {
				log.Errorf("failed to remove srv %s, err: %v", nodeID, err)
				return err
			}

			log.Infof("node %s left successfully", nodeID)
			return nil
		}
	}

	log.Infof("node %s not exists in raft group", nodeID)
	return nil
}

func (c Core) Apply(msg []byte) (err error) {
	if !c.isLeader() {
		log.
			Warn("Apply request should not be executed by nonleader node")

		return ErrNotLeader
	}

	f := c.raft.Apply(msg, 10*time.Second)
	if err = f.Error(); err != nil {
		log.
			WithFields(log.Fields{
				"msg": msg,
			}).
			Errorf("Core.watchLeaderChanges applyTo raft failed: %v", f.Error())
	}

	return
}

// watchContainerChanges would load container in detail and recalculate its checksum. If old and new is different
// trigger watcher.IWatcher to notify changes and persistence.Repository to update new checksum.
//
// Notice: only leader would trigger this logic.
func (c Core) watchContainerChanges(ns, key string) error {
	log.WithFields(log.Fields{
		"ns":  ns,
		"key": key,
	}).Debug("Core.watchContainerChanges called")

	container, err := c.GetContainer(key, ns)
	if err != nil {
		return errors.Wrap(err, "Core.watchContainerChanges failed to c.repo.Converter().ToContainer(v)")
	}

	oldCheckSum := container.CheckSum("")
	content, _ := json.Marshal(container)
	newCheckSum := container.CheckSum(hash.CheckSum(content))

	log.
		WithFields(log.Fields{
			"old": oldCheckSum,
			"new": newCheckSum,
		}).
		Debug("comparing container's checksum")

	if oldCheckSum == newCheckSum {
		// no changes happened, do nothing.
		return nil
	}

	if err = c.repo.UpdateContainerCheckSum(ns, key, newCheckSum); err != nil {
		return errors.Wrapf(err, "Core.watchContainerChanges failed to c.repo.UpdateContainerChecksum")
	}

	// now update container's checksum into persistence
	if oldCheckSum == "" {
		// would not trigger changes, only update container's checksum
		return nil
	}

	// FIXED(@yeqwon) reset cache container cache, A brute force to delete all ns+key+formats(TOML/JSON)
	go func() {
		// container in JSON format changes and delete cache
		c.watcher.ChangeNotify(watcher.Changes{
			CheckSum:  newCheckSum,
			Key:       key,
			Namespace: ns,
			Format:    datatypes.JSON,
			Data:      content,
		})

		// DONE(@yeqown): reset cache
		cacheKey := c.genContainerCacheKey(ns, key, datatypes.JSON)
		c.delContainerCache(cacheKey)
	}()

	go func() {
		// container in TOML format changes and delete cache
		c.watcher.ChangeNotify(watcher.Changes{
			CheckSum:  newCheckSum,
			Key:       key,
			Namespace: ns,
			Format:    datatypes.TOML,
			Data:      content,
		})

		cacheKey := c.genContainerCacheKey(ns, key, datatypes.TOML)
		c.delContainerCache(cacheKey)
	}()

	return nil
}

// isLeader only return true if current node is leader.
func (c Core) isLeader() bool {
	return c.raft.State() == raft.Leader
}

func (c Core) ShouldForwardToLeader() (shouldForward bool, leadAddr string) {
	return !c.isLeader(), c.fsm.LeaderAddr()
}
