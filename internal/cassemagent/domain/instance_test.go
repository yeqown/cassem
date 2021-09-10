package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/yeqown/cassem/api/concept"
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
		_ = t.pool.Register(insId, "app", "env", []string{"key1", "key2"})
		t.pool.Unregister(insId)
	}
	t.Equal(0, len(t.pool.(*instancePool).instances))

	_ = t.pool.Register(insId, "app", "env", []string{"key1", "key2"})
	t.Equal(1, len(t.pool.(*instancePool).instances))
	t.T().Logf("%+v", t.pool)

	insIds := t.pool.ListWatchingInstances("app", "env", "key1")
	t.Contains(insIds, "insId")
	insIds = t.pool.ListWatchingInstances("app", "env", "key2")
	t.Contains(insIds, "insId")
	insIds = t.pool.ListWatchingInstances("app", "env", "key3")
	t.NotContains(insIds, "insId")
}

func (t testInstancePoolSuite) Test_Notify() {
	ins1 := "ins1"
	ch := t.pool.Register(ins1, "app", "env", []string{""})
	_ = t.pool.Register(ins1, "app", "env", []string{"key1"})
	actualCnt := 0

	go func() {
		for range ch {
			actualCnt++
		}
	}()

	insIds := t.pool.ListWatchingInstances("app", "env", "key1")
	t.Contains(insIds, ins1)

	// could not receive
	t.pool.Notify(ins1, nil)

	// could not receive
	t.pool.Notify("ins2", &concept.Element{
		Metadata: &concept.ElementMetadata{
			Key:                "key1",
			App:                "app",
			Env:                "env",
			LatestVersion:      0,
			UnpublishedVersion: 0,
			UsingVersion:       0,
			UsingFingerprint:   "",
			ContentType:        0,
		},
		Raw:     nil,
		Version: 0,
	})

	// could receive
	t.pool.Notify(ins1, &concept.Element{
		Metadata: &concept.ElementMetadata{
			Key:                "key1",
			App:                "app",
			Env:                "env",
			LatestVersion:      0,
			UnpublishedVersion: 0,
			UsingVersion:       0,
			UsingFingerprint:   "",
			ContentType:        0,
		},
		Raw:     []byte("ins1"),
		Version: 0,
	})
	t.pool.Unregister(ins1)
	time.Sleep(2 * time.Second)
	t.Equal(1, actualCnt)
}

func Test_instancePool(t *testing.T) {
	suite.Run(t, new(testInstancePoolSuite))
}
