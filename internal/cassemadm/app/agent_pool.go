package app

// The purpose of agent_pool.go is that helps app to publish elements to
// cassem ap, so that agent can update local cache, then agent notify all clients
// those are watching the element.

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/set"
)

// agentPool manages all ap those registered in cassemdb.
type agentPool struct {
	// nodes indicates map[agentId]*agentNode
	nodes map[string]*agentNode
	// allAgentIds contains all agent ids, it is maintained as nodes.
	allAgentIds set.StringSet
	// rwMutex protects goroutines accessing nodes.
	rwMutex sync.RWMutex

	agg concept.AgentHybrid
	// once make sure agentPool.run will be called only once.
	once sync.Once
}

// newAgentPool construct a agentPool instance and automatically run routines.
func newAgentPool(agg concept.AgentHybrid) *agentPool {
	p := &agentPool{
		nodes:       make(map[string]*agentNode, 16),
		allAgentIds: set.NewStringSet(16),
		rwMutex:     sync.RWMutex{},
		agg:         agg,
		once:        sync.Once{},
	}

	p.run()

	return p
}

// run to start background routines to help agentPool manage agent instances.
func (p *agentPool) run() {
	p.once.Do(func() {
		ch := make(chan *concept.AgentInstanceChange, _SIZE_AGENT_NODE_BUF)
		runtime.GoFunc("watchingAgentInstanceRaw", func() error {
			return p.agg.Watch(context.Background(), ch)
		})
		runtime.GoFunc("updateAgentInstance", p.updateAgentInstanceFromCh(ch))
	})

	// update all ap firstly while cassem adm starting,
	// in case of which adm recover from panic or exception shutdown.
	if err := p.updateAgentNodesManually(); err != nil {
		log.
			WithFields(log.Fields{"error": err}).
			Warn("agentPool.run failed to updateAgentNodesManually")
	}
}

func (p *agentPool) all() []*concept.AgentInstance {
	out := make([]*concept.AgentInstance, 0, len(p.nodes))
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	for key, v := range p.nodes {
		if v == nil {
			continue
		}
		out = append(out, p.nodes[key].AgentInstance)
	}

	return out
}

// DONE(@yeqown): update agent nodes manually at the start of the agent pool.
func (p *agentPool) updateAgentNodesManually() error {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	r, err := p.agg.GetAgents(ctx, "", 100)
	if err != nil {
		return errors.Wrap(err, "agentPool.updateAgentNodesManually")
	}

	if r.HasMore {
		log.
			Warn("agentPool.updateAgentNodesManually can only handling 1000 ap.")
	}

	log.
		WithFields(log.Fields{
			"nodeCount": len(r.Agents),
			"hasMore":   r.HasMore,
			"nextSeek":  r.NextSeek,
		}).
		Debug("agentPool.updateAgentNodesManually called")

	// update whole ap
	p.rwMutex.Lock()
	for idx, v := range r.Agents {
		no, ok := p.nodes[v.GetAgentId()]
		if !ok {
			p.nodes[v.GetAgentId()] = newAgentNode(r.Agents[idx])
		} else {
			no.updateAddr(v.GetAddr())
		}
	}
	p.rwMutex.Unlock()

	return nil
}

func (p *agentPool) updateAgentInstanceFromCh(ch <-chan *concept.AgentInstanceChange) func() error {
	return func() error {
		// There is a node changed, then judge and update p.nodes and p.allAgentIds
		for change := range ch {
			if change == nil || change.GetIns() == nil {
				continue
			}

			agentId := change.GetIns().GetAgentId()
			agentAddr := change.GetIns().GetAddr()
			switch change.Op {
			case concept.ChangeOp_NEW, concept.ChangeOp_UPDATE:
				p.rwMutex.Lock()
				node, ok := p.nodes[agentId]
				if !ok {
					// new node
					node = newAgentNode(change.GetIns())
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

// notifyAll dispatch element to all ap.
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
	*concept.AgentInstance

	ch chan *concept.Element
	c  agent.DeliveryClient
}

// _SIZE_AGENT_NODE_BUF buf size of agent node notify channel, it indicates the maximum
// elements could be held by agent node.
const _SIZE_AGENT_NODE_BUF = 1024

func newAgentNode(ins *concept.AgentInstance) *agentNode {
	if ins == nil || ins.GetAddr() == "" {
		panic("")
	}

	u, err := url.Parse(ins.GetAddr())
	if err != nil {
		log.
			WithField("error", err).
			Warn("cassemadm.newAgentNode failed parse addr")
	}
	_ = u

	return &agentNode{
		AgentInstance: ins,
		ch:            make(chan *concept.Element, _SIZE_AGENT_NODE_BUF),
		c:             nil,
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
				if len(batch) == 0 {
					continue
				}

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
			"agentId":   n.GetAgentId(),
		}).
		Debug("agentNode.delivery called")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// TODO(@yeqown): support dispatch to specific instance
	req := &agent.DispatchReq{
		Elems: batch,
	}
	_, err := n.getClient().Dispatch(timeoutCtx, req)
	if err != nil {
		// message missed
		log.
			WithFields(log.Fields{
				"req":     req,
				"error":   err,
				"agentId": n.GetAgentId(),
			}).
			Warn("agentNode.delivery failed dispatch")
	}
}

// FIXED(@yeqown): shouldn't retry forever: maxRetryCount = 3
func (n *agentNode) getClient() agent.DeliveryClient {
	if n.c != nil {
		return n.c
	}

	var (
		err      error
		retryCnt int
	)
retry:
	n.c, err = agent.DialDelivery(n.Addr)
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

	// FIXME(@yeqown): maybe data race
	n.Addr = addr
	n.c = nil
}
