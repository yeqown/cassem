package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/yeqown/cassem/concept"
)

type testInstancePoolSuite struct {
	suite.Suite

	pool InstancePool
}

func (t *testInstancePoolSuite) SetupSuite() {
	t.pool = NewInstancePool()
}

func (t testInstancePoolSuite) Test_Register_Unregister() {
	var insId = "insId"
	for i := 0; i < 1000; i++ {
		t.pool.Register(insId)
		t.pool.Unregister(insId)
	}

	t.Equal(0, len(t.pool.(*instancePool).instances))
}

func (t testInstancePoolSuite) Test_Notify() {
	ins1 := "ins1"
	ch := t.pool.Register(ins1)
	actualCnt := 0

	go func() {
		for range ch {
			actualCnt++
		}
	}()

	// could not receive
	t.pool.Notify(ins1, nil)

	// could not receive
	t.pool.Notify("ins2", &concept.Element{
		Metadata: nil,
		Raw:      nil,
		Version:  0,
	})

	// could receive
	t.pool.Notify(ins1, &concept.Element{
		Metadata: nil,
		Raw:      []byte("ins1"),
		Version:  0,
	})
	t.pool.Unregister(ins1)
	time.Sleep(2 * time.Second)
	t.Equal(1, actualCnt)
}

func Test_instancePool(t *testing.T) {
	suite.Run(t, new(testInstancePoolSuite))
}
