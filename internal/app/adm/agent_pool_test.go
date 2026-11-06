package adm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	pb "github.com/yeqown/cassem/internal/protogen"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/yeqown/cassem/internal/concept"
	"google.golang.org/grpc"
)

type fakeDeliveryClient struct{}

func (fakeDeliveryClient) Dispatch(ctx context.Context, in *pb.DispatchReq, opts ...grpc.CallOption) (*pb.DispatchResp, error) {
	return &pb.DispatchResp{}, nil
}

var _ pb.DeliveryClient = fakeDeliveryClient{}

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
	ch := make(chan *pb.AgentInstanceChange, 10)
	ins := &pb.AgentInstance{
		AgentId: "agentId",
		Addr:    "addr",
	}

	done := make(chan struct{})
	fn := t.ap.updateAgentInstanceFromCh(ch)
	go func() {
		defer close(done)
		_ = fn()
	}()
	defer func() {
		close(ch)
		<-done
	}()

	ch <- &pb.AgentInstanceChange{Ins: ins, Op: pb.ChangeOp_NEW}
	t.eventually(func() bool {
		t.ap.rwMutex.RLock()
		defer t.ap.rwMutex.RUnlock()
		return len(t.ap.allAgentIds) == 1 && len(t.ap.nodes) == 1
	})

	updated := &pb.AgentInstance{AgentId: ins.AgentId, Addr: "updated-addr"}
	ch <- &pb.AgentInstanceChange{Ins: updated, Op: pb.ChangeOp_UPDATE}
	t.eventually(func() bool {
		t.ap.rwMutex.RLock()
		defer t.ap.rwMutex.RUnlock()
		node, ok := t.ap.nodes[ins.AgentId]
		return len(t.ap.allAgentIds) == 1 && len(t.ap.nodes) == 1 && ok && node.snapshot().Addr == updated.Addr
	})

	ch <- &pb.AgentInstanceChange{Ins: updated, Op: pb.ChangeOp_DELETE}
	t.eventually(func() bool {
		t.ap.rwMutex.RLock()
		defer t.ap.rwMutex.RUnlock()
		return len(t.ap.allAgentIds) == 0 && len(t.ap.nodes) == 0
	})

	ch <- &pb.AgentInstanceChange{Ins: ins, Op: pb.ChangeOp_NEW}
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
	node, err := newAgentNode(&pb.AgentInstance{AgentId: "agent", Addr: "127.0.0.1:1"})
	require.NoError(t, err)
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
		AgentInstance: &pb.AgentInstance{
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
	assert.Equal(t, "cn", node.Annotations["zone"])
	assert.NotContains(t, node.Annotations, "new")
}

func TestNewAgentNodeRejectsInvalidInstance(t *testing.T) {
	_, err := newAgentNode(nil)
	require.Error(t, err)

	_, err = newAgentNode(&pb.AgentInstance{AgentId: "agent"})
	require.Error(t, err)
}

func TestAgentNodeUpdateInstance(t *testing.T) {
	client := fakeDeliveryClient{}
	node := &agentNode{
		AgentInstance: &pb.AgentInstance{AgentId: "agent-1", Addr: "127.0.0.1:9000"},
		mu:            sync.RWMutex{},
		c:             client,
	}

	node.updateInstance(&pb.AgentInstance{AgentId: "agent-1", Addr: "127.0.0.1:9000"})
	assert.Equal(t, client, node.c)

	node.updateInstance(&pb.AgentInstance{AgentId: "agent-1", Addr: "127.0.0.1:9001"})
	assert.Nil(t, node.c)
	assert.Equal(t, "127.0.0.1:9001", node.Addr)

	node.updateInstance(nil)
	assert.Equal(t, "127.0.0.1:9001", node.Addr)
}

func Test_AgentPool(t *testing.T) {
	suite.Run(t, new(testAgentPoolSuite))
}
