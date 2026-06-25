package watcher

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testObserver struct {
	id     string
	topics []string
	ch     chan IChange
	once   sync.Once
	closed chan struct{}
}

func newTestObserver(id string, topics []string, buffer int) *testObserver {
	return &testObserver{
		id:     id,
		topics: topics,
		ch:     make(chan IChange, buffer),
		closed: make(chan struct{}),
	}
}

func (t *testObserver) Identity() string         { return t.id }
func (t *testObserver) Outbound() <-chan IChange { return t.ch }
func (t *testObserver) Inbound() chan<- IChange  { return t.ch }
func (t *testObserver) Topics() []string         { return t.topics }
func (t *testObserver) Close() {
	t.once.Do(func() {
		close(t.closed)
		close(t.ch)
	})
}

type testChange struct {
	topic string
}

func (t testChange) Topic() string    { return t.topic }
func (t testChange) Type() ChangeType { return ChangeType_KV }

func TestTopicBucketDistributeDeliversToObserver(t *testing.T) {
	bucket := newTopicBucket()
	observer := newTestObserver("fast", []string{"topic"}, 1)
	bucket.add(observer)

	notify := testChange{topic: "topic"}
	bucket.distribute(notify)

	select {
	case got := <-observer.Outbound():
		require.Equal(t, notify.Topic(), got.Topic())
	case <-time.After(time.Second):
		t.Fatal("expected notify to reach observer")
	}
}

func TestTopicBucketDistributeRemovesSlowObserver(t *testing.T) {
	bucket := newTopicBucket()
	slow := newTestObserver("slow", []string{"topic"}, 0)
	bucket.add(slow)

	start := time.Now()
	bucket.distribute(testChange{topic: "topic"})
	require.GreaterOrEqual(t, time.Since(start), slowObserverTimeout)

	select {
	case <-slow.closed:
	case <-time.After(time.Second):
		t.Fatal("expected slow observer to be closed")
	}

	bucket.RLock()
	defer bucket.RUnlock()
	require.Empty(t, bucket.observers)
}

func TestChannelWatcherChangeNotifyBlocksUntilReceiverReady(t *testing.T) {
	w := &channelWatcher{ch: make(chan IChange)}
	done := make(chan struct{})

	go func() {
		w.ChangeNotify(testChange{topic: "topic"})
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("expected ChangeNotify to block without receiver")
	case <-time.After(50 * time.Millisecond):
	}

	select {
	case got := <-w.ch:
		require.Equal(t, "topic", got.Topic())
	case <-time.After(time.Second):
		t.Fatal("expected notify from channel watcher")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected ChangeNotify to finish after receiver reads")
	}
}
