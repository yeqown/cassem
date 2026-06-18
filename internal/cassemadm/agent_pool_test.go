package cassemadm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
)

type fakeDeliveryClient struct{}

func (fakeDeliveryClient) Dispatch(ctx context.Context, in *agent.DispatchReq, opts ...grpc.CallOption) (*agent.DispatchResp, error) {
	return &agent.DispatchResp{}, nil
}

var _ agent.DeliveryClient = fakeDeliveryClient{}

type testAgentPoolSuite struct {
	suite.Suite

	ap *agentPool
}

func (t *testAgentPoolSuite) SetupTest() {
	t.ap = newAgentPool(nil)
}

func (t *testAgentPoolSuite) eventually(assertion func() bool) {
	t.Require().Eventually(assertion, time.Second, 10*time.Millisecond)
}

func (t *testAgentPoolSuite) Test_consumeAgentInstanceChange() {
	ch := make(chan *concept.AgentInstanceChange, 10)
	defer close(ch)

	change := &concept.AgentInstanceChange{
		Ins: &concept.AgentInstance{
			AgentId:     "agentId",
			Addr:        "addr",
			Annotations: nil,
		},
		Op: concept.ChangeOp_NEW,
	}

	go func() {
		fn := t.ap.updateAgentInstanceFromCh(ch)
		_ = fn()
	}()

	ch <- change
	t.eventually(func() bool {
		t.ap.rwMutex.RLock()
		defer t.ap.rwMutex.RUnlock()
		return len(t.ap.allAgentIds) == 1 && len(t.ap.nodes) == 1
	})

	change.Op = concept.ChangeOp_UPDATE
	ch <- change
	t.eventually(func() bool {
		t.ap.rwMutex.RLock()
		defer t.ap.rwMutex.RUnlock()
		return len(t.ap.allAgentIds) == 1 && len(t.ap.nodes) == 1
	})

	change.Op = concept.ChangeOp_DELETE
	ch <- change
	t.eventually(func() bool {
		t.ap.rwMutex.RLock()
		defer t.ap.rwMutex.RUnlock()
		return len(t.ap.allAgentIds) == 0 && len(t.ap.nodes) == 0
	})

	change.Op = concept.ChangeOp_NEW
	ch <- change
	t.eventually(func() bool {
		t.ap.rwMutex.RLock()
		defer t.ap.rwMutex.RUnlock()
		return len(t.ap.allAgentIds) == 1 && len(t.ap.nodes) == 1
	})
}

func (t *testAgentPoolSuite) Test_agentNode_zip() {
	err := t.ap.notifyAgent(&concept.Element{
		Metadata:  nil,
		Raw:       []byte("this is raw"),
		Version:   1,
		Published: false,
	}, "agentId")
	t.Require().NoError(err)
}

func TestAgentPoolConcurrentAccess(t *testing.T) {
	ap := newAgentPool(nil)
	node := newAgentNode(&concept.AgentInstance{AgentId: "agent", Addr: "127.0.0.1:1"})
	ap.nodes["agent"] = node
	ap.allAgentIds["agent"] = struct{}{}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			node.updateAddr(fmt.Sprintf("127.0.0.1:%d", i+1))
			_ = node.snapshot()
			_ = ap.all()
			_ = ap.getAgentIdKeys()
			_ = ap.notifyAgentInstances(&concept.Element{Version: int32(i)}, []string{"ins"}, "agent")
		}(i)
	}
	wg.Wait()

	require.Len(t, ap.getAgentIdKeys(), 1)
}

func TestNormalizeInstanceIds(t *testing.T) {
	ids := []string{"ins-3", "ins-1", "ins-2"}

	got := normalizeInstanceIds(ids)

	assert.Equal(t, "ins-1,ins-2,ins-3", got)
	assert.Equal(t, []string{"ins-3", "ins-1", "ins-2"}, ids)
	assert.Empty(t, normalizeInstanceIds(nil))
}

func TestAgentNodeSnapshotCopiesAnnotations(t *testing.T) {
	node := &agentNode{
		AgentInstance: &concept.AgentInstance{
			AgentId:     "agent-1",
			Addr:        "127.0.0.1:9000",
			Annotations: map[string]string{"zone": "cn"},
		},
		mu: sync.RWMutex{},
	}

	snapshot := node.snapshot()
	snapshot.Annotations["zone"] = "us"
	snapshot.Annotations["new"] = "value"

	assert.Equal(t, "agent-1", snapshot.AgentId)
	assert.Equal(t, "127.0.0.1:9000", snapshot.Addr)
	assert.Equal(t, "cn", node.AgentInstance.Annotations["zone"])
	assert.NotContains(t, node.AgentInstance.Annotations, "new")
}

func TestAgentNodeUpdateInstance(t *testing.T) {
	client := fakeDeliveryClient{}
	node := &agentNode{
		AgentInstance: &concept.AgentInstance{AgentId: "agent-1", Addr: "127.0.0.1:9000"},
		mu:            sync.RWMutex{},
		c:             client,
	}

	node.updateInstance(&concept.AgentInstance{AgentId: "agent-1", Addr: "127.0.0.1:9000"})
	assert.Equal(t, client, node.c)

	node.updateInstance(&concept.AgentInstance{AgentId: "agent-1", Addr: "127.0.0.1:9001"})
	assert.Nil(t, node.c)
	assert.Equal(t, "127.0.0.1:9001", node.AgentInstance.Addr)

	node.updateInstance(nil)
	assert.Equal(t, "127.0.0.1:9001", node.AgentInstance.Addr)
}

func Test_AgentPool(t *testing.T) {
	suite.Run(t, new(testAgentPoolSuite))
}
