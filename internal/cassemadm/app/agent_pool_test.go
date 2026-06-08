package app

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/yeqown/cassem/api/concept"
)

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

func Test_AgentPool(t *testing.T) {
	suite.Run(t, new(testAgentPoolSuite))
}
