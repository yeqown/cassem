package adm

// The purpose of agent_pool.go is that helps app to publish elements to
// cassem ap, so that agent can update local cache, then agent notify all clients
// those are watching the element.

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/runtime"
)

// agentPool manages all ap those registered in cassemdb.
type agentPool struct {
	// nodes indicates map[agentId]*agentNode
	nodes map[string]*agentNode
	// allAgentIds contains all agent ids, it is maintained as nodes.
	allAgentIds map[string]struct{}
	// rwMutex protects goroutines accessing nodes.
	rwMutex sync.RWMutex

	agg    concept.AgentHybrid
	ctx    context.Context
	cancel context.CancelFunc
	// once make sure agentPool.run will be called only once.
	once sync.Once
}

// newAgentPool construct a agentPool instance and automatically run routines.
func newAgentPool(agg concept.AgentHybrid) *agentPool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &agentPool{
		nodes:       make(map[string]*agentNode, 16),
		allAgentIds: make(map[string]struct{}, 16),
		rwMutex:     sync.RWMutex{},
		agg:         agg,
		ctx:         ctx,
		cancel:      cancel,
		once:        sync.Once{},
	}

	if agg != nil {
		p.run()
	}

	return p
}

// run to start background routines to help agentPool manage agent instances.
func (p *agentPool) run() {
	p.once.Do(func() {
		ch := make(chan *concept.AgentInstanceChange, _SIZE_AGENT_NODE_BUF)
		runtime.GoFunc("watchingAgentInstanceRaw", func() error {
			return p.agg.Watch(p.ctx, ch)
		})
		runtime.GoFunc("updateAgentInstance", p.updateAgentInstanceFromCh(ch))
		runtime.GoFunc("updateAgentNodesManually.cron", func() error {
			ticker := time.NewTicker(45 * time.Second)
			defer ticker.Stop()
			for {
				// update all ap firstly while cassem adm starting,
				// in case of which adm recover from panic or exception shutdown.
				if err := p.updateAgentNodesManually(); err != nil {
					log.
						WithFields(log.Fields{"error": err}).
						Warn("agentPool.run failed to updateAgentNodesManually")
				}
				select {
				case <-ticker.C:
				case <-p.ctx.Done():
					return p.ctx.Err()
				}
			}
			// panic("impossible")
		})
	})
}

func (p *agentPool) all() []*concept.AgentInstance {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	out := make([]*concept.AgentInstance, 0, len(p.nodes))
	for _, node := range p.nodes {
		if node == nil {
			continue
		}
		out = append(out, node.snapshot())
	}

	return out
}

// DONE(@yeqown): update agent nodes manually at the start of the agent pool.
func (p *agentPool) updateAgentNodesManually() error {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	r, err := p.agg.GetAgents(ctx, "", 100)
	if err != nil {
		return fmt.Errorf("agentPool.updateAgentNodesManually: %w", err)
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

	newNodes := make(map[string]*agentNode, len(r.Agents))
	newAgentIds := make(map[string]struct{}, len(r.Agents))

	p.rwMutex.RLock()
	existing := make(map[string]*agentNode, len(p.nodes))
	for agentId, node := range p.nodes {
		existing[agentId] = node
	}
	p.rwMutex.RUnlock()

	for _, ins := range r.Agents {
		agentId := ins.GetAgentId()
		node, ok := existing[agentId]
		if ok {
			node.updateInstance(ins)
		} else {
			node, err = newAgentNode(ins)
			if err != nil {
				log.WithFields(log.Fields{"agentId": agentId, "error": err}).Warn("agentPool.updateAgentNodesManually skip invalid agent")
				continue
			}
		}
		newNodes[agentId] = node
		newAgentIds[agentId] = struct{}{}
	}

	p.rwMutex.Lock()
	p.nodes = newNodes
	p.allAgentIds = newAgentIds
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
			switch change.Op {
			case concept.ChangeOp_NEW, concept.ChangeOp_UPDATE:
				p.rwMutex.Lock()
				node, ok := p.nodes[agentId]
				if !ok {
					// new node
					var err error
					node, err = newAgentNode(change.GetIns())
					if err != nil {
						log.WithFields(log.Fields{"agentId": agentId, "error": err}).Warn("agentPool.updateAgentInstanceFromCh skip invalid agent")
						p.rwMutex.Unlock()
						continue
					}
					p.nodes[agentId] = node
				} else {
					// node update
					node.updateInstance(change.GetIns())
				}
				p.allAgentIds[agentId] = struct{}{}
				p.rwMutex.Unlock()
			case concept.ChangeOp_DELETE:
				p.rwMutex.Lock()
				delete(p.nodes, agentId)
				delete(p.allAgentIds, agentId)
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
	keys := p.getAgentIdKeys()
	log.
		WithFields(log.Fields{
			"keys": keys,
		}).
		Debug("cassemadm.app.agent.notifyAll called")
	return p.notifyAgent(elem, keys...)
}

// getAgentIdKeys returns all agent ids as a slice.
func (p *agentPool) getAgentIdKeys() []string {
	p.rwMutex.RLock()
	defer p.rwMutex.RUnlock()

	keys := make([]string, 0, len(p.allAgentIds))
	for k := range p.allAgentIds {
		keys = append(keys, k)
	}
	return keys
}

// notifyAgent helps app notify agent by agent ids.
func (p *agentPool) notifyAgent(elem *concept.Element, agentIds ...string) error {
	return p.notifyAgentInstances(elem, nil, agentIds...)
}

func (p *agentPool) notifyAgentInstances(elem *concept.Element, instanceIds []string, agentIds ...string) error {
	log.
		WithFields(log.Fields{
			"elem":        elem,
			"agentIds":    agentIds,
			"instanceIds": instanceIds,
		}).
		Debug("cassemamd.app.agent.notifyAgentInstances called")

	if len(agentIds) == 0 {
		return nil
	}

	p.rwMutex.RLock()
	nodes := make(map[string]*agentNode, len(agentIds))
	for _, agentId := range agentIds {
		node, ok := p.nodes[agentId]
		if !ok {
			log.
				WithFields(log.Fields{"agentId": agentId}).
				Warn("cassemadm.app.agentPool failed to find agentNode")
			continue
		}
		nodes[agentId] = node
	}
	p.rwMutex.RUnlock()

	item := agentDispatchItem{
		elem:        elem,
		instanceIds: append([]string(nil), instanceIds...),
	}
	for agentId, node := range nodes {
		select {
		case node.postbox() <- item:
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

type agentDispatchItem struct {
	elem        *concept.Element
	instanceIds []string
}

// agentNode contains agent node information, includes the address to
// agent node.
type agentNode struct {
	*concept.AgentInstance

	mu sync.RWMutex
	ch chan agentDispatchItem
	c  agent.DeliveryClient
}

// _SIZE_AGENT_NODE_BUF buf size of agent node notify channel, it indicates the maximum
// elements could be held by agent node.
const _SIZE_AGENT_NODE_BUF = 1024

func newAgentNode(ins *concept.AgentInstance) (*agentNode, error) {
	if ins == nil {
		return nil, fmt.Errorf("agent instance is required")
	}
	if ins.GetAddr() == "" {
		return nil, fmt.Errorf("agent addr is required")
	}

	u, err := url.Parse(ins.GetAddr())
	if err != nil {
		log.
			WithField("error", err).
			Warn("cassemadm.newAgentNode failed parse addr")
	}
	_ = u

	n := &agentNode{
		AgentInstance: ins,
		mu:            sync.RWMutex{},
		ch:            make(chan agentDispatchItem, _SIZE_AGENT_NODE_BUF),
		c:             nil,
	}

	n.run()

	return n, nil
}

func (n *agentNode) postbox() chan<- agentDispatchItem {
	return n.ch
}

func (n *agentNode) snapshot() *concept.AgentInstance {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.AgentInstance == nil {
		return nil
	}
	annotations := make(map[string]string, len(n.AgentInstance.GetAnnotations()))
	for k, v := range n.AgentInstance.GetAnnotations() {
		annotations[k] = v
	}
	return &concept.AgentInstance{
		AgentId:     n.AgentInstance.GetAgentId(),
		Addr:        n.AgentInstance.GetAddr(),
		Annotations: annotations,
	}
}

func (n *agentNode) agentId() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.AgentInstance == nil {
		return ""
	}
	return n.AgentInstance.GetAgentId()
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
			batch     []agentDispatchItem
			t         = time.NewTicker(_WAIT_DURATION)
			waitTimes = 0
		)

		// reset loop variables
		reset := func() {
			t.Reset(_WAIT_DURATION)
			waitTimes = 0
			batch = make([]agentDispatchItem, 0, _MAX_SIZE)
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
func (n *agentNode) delivery(batch []agentDispatchItem) {
	log.
		WithFields(log.Fields{
			"batchSize": len(batch),
			"agentId":   n.agentId(),
		}).
		Debug("agentNode.delivery called")

	grouped := make(map[string]*agent.DispatchReq, len(batch))
	for _, item := range batch {
		key := normalizeInstanceIds(item.instanceIds)
		req, ok := grouped[key]
		if !ok {
			req = &agent.DispatchReq{
				Elems:       make([]*concept.Element, 0, len(batch)),
				InstanceIds: append([]string(nil), item.instanceIds...),
			}
			grouped[key] = req
		}
		req.Elems = append(req.Elems, item.elem)
	}

	for _, req := range grouped {
		client := n.getClient()
		if client == nil {
			log.
				WithFields(log.Fields{"req": req, "agentId": n.agentId()}).
				Warn("agentNode.delivery skip dispatch without client")
			continue
		}
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := client.Dispatch(timeoutCtx, req)
		cancel()
		if err != nil {
			log.
				WithFields(log.Fields{
					"req":     req,
					"error":   err,
					"agentId": n.agentId(),
				}).
				Warn("agentNode.delivery failed dispatch")
		}
	}
}

func normalizeInstanceIds(instanceIds []string) string {
	if len(instanceIds) == 0 {
		return ""
	}

	ids := append([]string(nil), instanceIds...)
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

// FIXED(@yeqown): shouldn't retry forever: maxRetryCount = 3
func (n *agentNode) getClient() agent.DeliveryClient {
	n.mu.RLock()
	if n.c != nil {
		c := n.c
		n.mu.RUnlock()
		return c
	}
	n.mu.RUnlock()

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.c != nil {
		return n.c
	}

	addr := ""
	if n.AgentInstance != nil {
		addr = n.AgentInstance.GetAddr()
	}
	var (
		err      error
		retryCnt int
	)
	for {
		n.c, err = agent.DialDelivery(addr)
		if err == nil {
			return n.c
		}

		log.
			WithFields(log.Fields{
				"addr":  addr,
				"error": err,
			}).
			Error("agentNode.updateAddr re-init failed")
		if retryCnt >= 3 {
			return n.c
		}
		retryCnt++
		time.Sleep(time.Second)
	}
}

func (n *agentNode) updateInstance(ins *concept.AgentInstance) {
	if ins == nil {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.AgentInstance != nil && ins.GetAddr() != n.AgentInstance.GetAddr() {
		n.c = nil
	}
	n.AgentInstance = ins
}

func (n *agentNode) updateAddr(addr string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.AgentInstance != nil && addr == n.AgentInstance.GetAddr() {
		return
	}

	if n.AgentInstance == nil {
		n.AgentInstance = &concept.AgentInstance{}
	}
	n.AgentInstance.Addr = addr
	n.c = nil
}
