package domain

import (
	"strings"
	"sync"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/set"
)

// InstancePool is a pool provide open API ability to Register / Unregister / Notify instances.
type InstancePool interface {
	Register(insId string, app, env string, keys []string) <-chan *concept.Element

	Unregister(insId string)

	Notify(insId string, element *concept.Element)

	ListWatchingInstances(app, env, key string) set.StringSet
}

var (
	_ InstancePool = &instancePool{}
)

const (
	// _SIZE_BUF_ELEM means the maximum size of chan *concept.Element can hold.
	_SIZE_BUF_ELEM = 10
	// _SIZE_INIT_CAP means the initial capacity of instancePool's instances.
	_SIZE_INIT_CAP = 32
)

// instanceNode contains all fields to help dispatch.
type instanceNode struct {
	insId           string
	ch              chan *concept.Element
	watchingSetKeys []string
}

func newInstanceNode(insId string, ch chan *concept.Element, keys []string) *instanceNode {
	return &instanceNode{
		insId:           insId,
		ch:              ch,
		watchingSetKeys: keys,
	}
}

// instancePool manages all instances who have connected to the host agent, current server process.
// purpose of instancePool is to locate instance channel so that changes can be pushed to instance
// client, so instance to client must keep a tcp connections, make sure heartbeat mechanism is enabled.
//
// instancePool should maintain instance by itself, or memory leak will happen. make sure the instance would be
// removed from appPool while an instance is disconnected.
type instancePool struct {
	rwMutex sync.RWMutex

	// map[insId]*instanceNode
	instances map[string]*instanceNode

	// map[app-env-key]set.StringSet
	watchingSet map[string]set.StringSet
}

func NewInstancePool() InstancePool {
	return &instancePool{
		rwMutex:     sync.RWMutex{},
		instances:   make(map[string]*instanceNode, _SIZE_INIT_CAP),
		watchingSet: make(map[string]set.StringSet, _SIZE_INIT_CAP),
	}
}

func genWatchingKey(app, env, key string) string {
	return strings.Join([]string{app, env, key}, "-")
}

func (p *instancePool) Register(insId string, app, env string, watchingKeys []string) <-chan *concept.Element {
	keys := make([]string, len(watchingKeys))
	for idx, v := range watchingKeys {
		keys[idx] = genWatchingKey(app, env, v)
	}

	ch := p.register(insId, keys)
	return ch
}

func (p *instancePool) Unregister(insId string) {
	p.unregister(insId)
}

func (p *instancePool) ListWatchingInstances(app, env, key string) set.StringSet {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	k := strings.Join([]string{app, env, key}, "-")
	v, ok := p.watchingSet[k]
	if !ok {
		return nil
	}

	return v
}

func (p *instancePool) Notify(insId string, element *concept.Element) {
	if element == nil {
		return
	}

	p.push(insId, element)
}

func (p *instancePool) push(insId string, elem *concept.Element) {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	i, ok := p.instances[insId]
	if !ok {
		log.
			WithFields(log.Fields{
				"insId":   insId,
				"element": elem,
			}).
			Debug("instancePool could not locate instance")
		return
	}

	// nonblocking
	select {
	case i.ch <- elem:
	default:
	}

	return
}

func (p *instancePool) register(insId string, keys []string) <-chan *concept.Element {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	v, ok := p.instances[insId]
	if ok {
		return v.ch
	}

	ch := make(chan *concept.Element, _SIZE_BUF_ELEM)
	p.instances[insId] = newInstanceNode(insId, ch, keys)

	// add instance to watching keys.
	for _, key := range keys {
		s, ok := p.watchingSet[key]
		if !ok {
			s = set.NewStringSet(4)
		}
		s.Add(insId)
	}

	return ch
}

func (p *instancePool) unregister(insId string) {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	i, ok := p.instances[insId]
	if !ok {
		return
	}

	// remove instance to watching keys.
	for _, key := range i.watchingSetKeys {
		s, ok := p.watchingSet[key]
		if !ok {
			continue
		}
		s.Del(insId)
	}

	close(i.ch)
	delete(p.instances, insId)
}
