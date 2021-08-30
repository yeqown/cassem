package domain

import (
	"sync"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/concept"
)

// InstancePool is a pool provide open API ability to Register / Unregister / Notify instances.
type InstancePool interface {
	Register(insId string) <-chan *concept.Element

	Unregister(insId string)

	Notify(insId string, element *concept.Element)
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

// instancePool manages all instances who have connected to the host agent, current server process.
// purpose of instancePool is to locate instance channel so that changes can be pushed to instance
// client, so instance to client must keep a tcp connections, make sure heartbeat mechanism is enabled.
//
// instancePool should maintain instance by itself, or memory leak will happen. make sure the instance would be
// removed from appPool while an instance is disconnected.
type instancePool struct {
	rwMutex sync.RWMutex

	// map[insId]chan<-concept.Element
	instances map[string]chan *concept.Element
}

func NewInstancePool() InstancePool {
	return &instancePool{
		rwMutex:   sync.RWMutex{},
		instances: make(map[string]chan *concept.Element, _SIZE_INIT_CAP),
	}
}

func (p *instancePool) Register(insId string) <-chan *concept.Element {
	ch := p.register(insId)
	return ch
}

func (p *instancePool) Unregister(insId string) {
	p.unregister(insId)
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

	ch, ok := p.instances[insId]
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
	case ch <- elem:
	default:
	}

	return
}

func (p *instancePool) register(insId string) <-chan *concept.Element {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	v, ok := p.instances[insId]
	if ok {
		return v
	}

	ch := make(chan *concept.Element, _SIZE_BUF_ELEM)
	p.instances[insId] = ch
	return ch
}

func (p *instancePool) unregister(insId string) {
	p.rwMutex.Lock()
	defer p.rwMutex.Unlock()

	// FIXME(@yeqown): should close the channel first?
	delete(p.instances, insId)
}
