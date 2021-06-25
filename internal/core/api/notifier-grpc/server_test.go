package notifier_test

import (
	"context"
	"log"
	"testing"
	"time"

	notifier "github.com/yeqown/cassem/internal/core/api/notifier-grpc"
	pb "github.com/yeqown/cassem/internal/core/api/notifier-grpc/gen"
	"github.com/yeqown/cassem/internal/watcher"

	"google.golang.org/grpc"
)

func prepare() watcher.IWatcher {
	w := watcher.NewChannelWatcher(10)
	go func() {
		if err := notifier.ListenAndServe(":2020"); err != nil {
			log.Fatalf(err.Error())
		}
	}()

	return w
}

// ensure calling this after you started cassemagent.
func Test_Notifier_Watch(t *testing.T) {
	// start watcher and notifier
	w := prepare()

	// boot client
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cc, err := grpc.DialContext(timeoutCtx, "127.0.0.1:2020", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to connect server, err=%v", err)
	}
	req := pb.WatchReq{
		Watches: []*pb.WatchOption{
			{
				Namespace: "ns",
				Keys:      []string{"container-1"},
				Format:    pb.Format_JSON,
			},
			{
				Namespace: "ns",
				Keys:      []string{"container-1"},
				Format:    pb.Format_TOML,
			},
		},
	}
	stream, err := pb.NewWatcherClient(cc).Watch(context.TODO(), &req)
	if err != nil {
		t.Fatalf("failed to execute Watch, err=%v", err)
	}

	go func() {
		changes := new(pb.Changes)
		for {
			if err = stream.RecvMsg(changes); err != nil {
				t.Logf("failed to stream.RecvMsg(changes), err=%v", err)
				return
			}

			t.Logf("received: %+v", changes)
		}
	}()

	// wait for client ready
	time.Sleep(2 * time.Second)
	// send changes to watcher
	go func() {
		for count := 20; count > 0; count-- {
			w.ChangeNotify(watcher.Changes{
				Key:       "container-1",
				Namespace: "ns",
				Format:    "json",
				CheckSum:  "12312312",
				Data:      nil,
			})

			time.Sleep(100 * time.Millisecond)
		}
	}()

	time.Sleep(time.Second)
	t.Logf("cc.Close(): %v", cc.Close())

	// wait client receive messages, there's no need to sync wait.
	time.Sleep(2 * time.Second)
}

// ensure calling this after you started cassemagent.
func Test_CASSEM(t *testing.T) {
	// boot client
	//timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	cc, err := grpc.Dial(":2021", grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to connect server, err=%v", err)
	}
	req := pb.WatchReq{
		Watches: []*pb.WatchOption{
			{
				Namespace: "ns",
				Keys:      []string{"del-container-test"},
				Format:    pb.Format_JSON,
			},
			{
				Namespace: "ns",
				Keys:      []string{"del-container-test"},
				Format:    pb.Format_TOML,
			},
		},
	}
	stream, err := pb.NewWatcherClient(cc).Watch(context.TODO(), &req)
	if err != nil {
		t.Fatalf("failed to execute Watch, err=%v", err)
	}

	go func() {
		changes := new(pb.Changes)
		for {
			if err = stream.RecvMsg(changes); err != nil {
				t.Logf("failed to stream.RecvMsg(changes), err=%v", err)
				return
			}

			t.Logf("received: %+v", changes)
		}
	}()

	time.Sleep(30 * time.Second)
}
