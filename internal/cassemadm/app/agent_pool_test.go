package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/yeqown/cassem/api/concept"
)

type testAgentPoolSuite struct {
	suite.Suite

	ap *agentPool
}

func (t *testAgentPoolSuite) SetupSuite() {
	t.ap = newAgentPool(nil)
}

func (t *testAgentPoolSuite) Test_consumeAgentInstanceChange() {
	ch := make(chan *concept.AgentInstanceChange, 10)

	change := &concept.AgentInstanceChange{
		Ins: &concept.AgentInstance{
			AgentId:     "agentId",
			Addr:        "addr",
			Annotations: nil,
		},
		Op: concept.ChangeOp_NEW,
	}

	// start consuming change channel goroutine.
	go func() {
		fn := t.ap.updateAgentInstanceFromCh(ch)
		t.Nil(fn())
	}()

	// new
	ch <- change
	time.Sleep(10 * time.Millisecond)
	t.NotEmpty(t.ap.nodes)
	t.NotEmpty(t.ap.allAgentIds)
	t.Equal(1, len(t.ap.allAgentIds))
	t.Equal(1, len(t.ap.nodes))

	// update
	change.Op = concept.ChangeOp_UPDATE
	ch <- change
	time.Sleep(10 * time.Millisecond)
	t.NotEmpty(t.ap.nodes)
	t.NotEmpty(t.ap.allAgentIds)
	t.Equal(1, len(t.ap.allAgentIds))
	t.Equal(1, len(t.ap.nodes))

	// delete
	change.Op = concept.ChangeOp_DELETE
	ch <- change
	time.Sleep(10 * time.Millisecond)
	t.Empty(t.ap.nodes)
	t.Empty(t.ap.allAgentIds)
	t.Equal(0, len(t.ap.allAgentIds))
	t.Equal(0, len(t.ap.nodes))

	// new
	ch <- change
	time.Sleep(10 * time.Millisecond)
	t.NotEmpty(t.ap.nodes)
	t.NotEmpty(t.ap.allAgentIds)
	t.Equal(1, len(t.ap.allAgentIds))
	t.Equal(1, len(t.ap.nodes))
}

func (t testAgentPoolSuite) Test_agentNode_zip() {
	err := t.ap.notifyAgent(&concept.Element{
		Metadata:  nil,
		Raw:       []byte("this is raw"),
		Version:   1,
		Published: false,
	}, "agentId")
	t.Require().NoError(err)
}

func Test_AgentPool(t *testing.T) {
	suite.Run(t, new(testAgentPoolSuite))
}
