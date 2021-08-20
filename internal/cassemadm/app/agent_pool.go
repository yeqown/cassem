package app

// The purpose of agent_pool.go is that helps app to publish elements to
// cassem agents, so that agent can update local cache, then agent notify all clients
// those are watching the element.

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/yeqown/log"

	apiagent "github.com/yeqown/cassem/internal/cassemagent/api"
	"github.com/yeqown/cassem/internal/concept"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/set"
)

// agentPool manages all agents those registered in cassemdb.
type agentPool struct {
	// nodes indicates map[agentId]*agentNode
	nodes map[string]*agentNode
	// allAgentIds contains all agent ids, it is maintained as nodes.
	allAgentIds set.StringSet
	// rwMutex protects goroutines accessing nodes.
	rwMutex sync.RWMutex

	agg concept.AgentHybrid
}

func newAgentPool() *agentPool {
	return &agentPool{
		nodes:       make(map[string]*agentNode, 16),
		allAgentIds: set.NewStringSet(16),
		rwMutex:     sync.RWMutex{},
	}
}

// run to start
func (p *agentPool) run() {
	ch := make(chan *concept.AgentInstanceChange, _SIZE_AGENT_NODE_BUF)
	runtime.GoFunc("watchingAgentInstanceRaw", func() error {
		return p.agg.Watch(context.Background(), ch)
	})
	runtime.GoFunc("updateAgentInstance", p.updateAgentInstanceFromCh(ch))
}

func (p *agentPool) updateAgentInstanceFromCh(ch <-chan *concept.AgentInstanceChange) func() error {
	return func() error {
		// There is a node changed, then judge and update p.nodes and p.allAgentIds
		for change := range ch {
			agentId := change.GetIns().GetAgentId()
			agentAddr := change.GetIns().GetAddr()
			switch change.Op {
			case concept.ChangeOp_NEW, concept.ChangeOp_UPDATE:
				p.rwMutex.Lock()
				node, ok := p.nodes[agentId]
				if !ok {
					// new node
					node = newAgentNode(agentId, agentAddr)
					p.nodes[agentId] = node
					node.run()
				} else {
					// node update
					node.updateAddr(agentAddr)
				}
				p.allAgentIds.Add(agentId)
				p.rwMutex.Unlock()
			case concept.ChangeOp_DELETE:
				p.rwMutex.Lock()
				delete(p.nodes, agentId)
				p.allAgentIds.Del(agentId)
				p.rwMutex.Unlock()
			default:
				continue
			}
		}
		return errors.New("watch channel closed")
	}
}

// notifyAll dispatch element to all agents.
func (p *agentPool) notifyAll(elem *concept.Element) error {
	return p.notifyAgent(elem, p.allAgentIds.Keys()...)
}

// notifyAgent helps app notify agent by agent ids.
func (p *agentPool) notifyAgent(elem *concept.Element, agentIds ...string) error {
	log.
		WithFields(log.Fields{
			"elem":     elem,
			"agentIds": agentIds,
		}).
		Debug("cassemamd.app.agent.notifyAgent called")

	if len(agentIds) == 0 {
		return nil
	}

	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()
	for _, agentId := range agentIds {
		node, ok := p.nodes[agentId]
		if !ok {
			continue
		}

		// nonblocking post to channel.
		select {
		case node.postbox() <- elem:
		default:
			log.
				WithFields(log.Fields{
					"elem":    elem,
					"agentId": agentId,
				}).
				Warn("cassemadm.app.agentPool skip one notify")
		}
	}

	return nil
}

// agentNode contains agent node information, includes the address to
// agent node.
type agentNode struct {
	Id   string
	Addr string
	ch   chan *concept.Element
	c    apiagent.DeliveryClient
}

// _SIZE_AGENT_NODE_BUF buf size of agent node notify channel, it indicates the maximum
// elements could be held by agent node.
const _SIZE_AGENT_NODE_BUF = 1024

func newAgentNode(id string, addr string) *agentNode {
	if addr == "" {
		panic("")
	}

	u, err := url.Parse(addr)
	if err != nil {
		panic(err)
	}
	_ = u

	return &agentNode{
		Id:   id,
		Addr: addr,
		ch:   make(chan *concept.Element, _SIZE_AGENT_NODE_BUF),
		c:    nil,
	}
}

func (n *agentNode) postbox() chan<- *concept.Element {
	return n.ch
}

// run starts a new goroutine to consume agent node's channel. delivery goroutine
// will package messages in 100ms, the maximum wait time is one time.Second.
// Or batch's size reached 100 or bigger.
//
// DONE(@yeqown): merge notify messages (fixed time and max size)
func (n *agentNode) run() {
	_MAX_SIZE := 100
	_WAIT_DURATION := 100 * time.Millisecond

	// delivery is a goroutine for agent node to consume agentNode.ch(channel signal)
	delivery := func() error {
		var (
			batch     []*concept.Element
			t         = time.NewTicker(_WAIT_DURATION)
			waitTimes = 0
		)

		// reset loop variables
		reset := func() {
			t.Reset(_WAIT_DURATION)
			waitTimes = 0
			batch = make([]*concept.Element, 0, _MAX_SIZE)
		}

	loop:
		for {
			select {
			case ele, ok := <-n.ch:
				log.
					WithFields(log.Fields{
						"elem": ele,
						"ok":   ok,
					}).
					Debug("agentNode.run.delivery routine consume one signal")
				if !ok {
					// if channel is closed, quit loop.
					break loop
				}
				batch = append(batch, ele)
				// wait again since a new message are received in wait(100ms) period,
				// so waitTimes increases, and reset ticker
				waitTimes++
				t.Reset(_WAIT_DURATION)
				// size limit: 100 or max waitTime limit 10 (10*100ms=1second)
				if len(batch) >= _MAX_SIZE || waitTimes >= 10 {
					n.delivery(batch)
					reset()
				}
			case <-t.C:
				// if there's no more message in the period of validity of the t.
				n.delivery(batch)
				reset()
			}
		}

		log.Debug("agentNode consume goroutine quit")
		return errors.New("agent consumes channel quit")
	}

	runtime.GoFunc("agentNode.run.delivery", delivery)
}

// delivery send dispatch request to agent.
func (n *agentNode) delivery(batch []*concept.Element) {
	log.
		WithFields(log.Fields{
			"batchSize": len(batch),
		}).
		Debug("agentNode.delivery called")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req := &apiagent.DispatchReq{
		Elems: batch,
	}
	_, err := n.getClient().Dispatch(timeoutCtx, req)
	if err != nil {
		// message missed
		log.
			WithFields(log.Fields{
				"req":   req,
				"error": err,
			}).
			Warn("agentNode.delivery failed dispatch")
	}
}

// FIXED(@yeqown): shouldn't retry forever: maxRetryCount = 3
func (n *agentNode) getClient() apiagent.DeliveryClient {
	if n.c != nil {
		return n.c
	}

	var (
		err      error
		retryCnt int
	)
retry:
	n.c, err = apiagent.DialDelivery(n.Addr)
	if err != nil {
		log.
			WithFields(log.Fields{
				"addr":  n.Addr,
				"error": err,
			}).
			Error("agentNode.updateAddr re-init failed")
		time.Sleep(time.Second)
		// maxRetryCount set to 3
		if retryCnt <= 3 {
			retryCnt++
			goto retry
		}
	}

	return n.c
}

func (n *agentNode) updateAddr(addr string) {
	if strings.Compare(addr, n.Addr) == 0 {
		return
	}

	// FIXME(@yeqown): data race
	n.Addr = addr
	n.c = nil
}
