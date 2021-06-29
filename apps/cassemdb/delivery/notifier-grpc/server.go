package notifier

//
//import (
//	"net"
//	"sync"
//
//	pb "github.com/yeqown/cassem/internal/core/api/notifier-grpc/gen"
//
//	"github.com/yeqown/cassem/internal/watcher"
//
//	"github.com/yeqown/log"
//	"google.golang.org/grpc"
//	"google.golang.org/grpc/reflection"
//)
//
//type Server struct {
//	//addr    string
//	watcher watcher.IWatcher
//	quit    chan struct{}
//}
//
//func (s Server) Watch(req *pb.WatchReq, stream pb.Watcher_WatchServer) (err error) {
//	opts := req.GetWatches()
//	changeCh := make(chan watcher.Changes, len(opts)*2)
//	once := sync.Once{}
//	closeFn := func() {
//		once.Do(func() {
//			close(changeCh)
//		})
//	}
//
//	obs := make([]watcher.IObserver, 0, len(opts))
//	for idx := range opts {
//		ob := genTopicObserver(changeCh, closeFn, opts[idx])
//		obs = append(obs, ob)
//	}
//
//	s.watcher.Subscribe(obs...)
//	defer func() {
//		// release resources of observer, PANIC would be executed?
//		log.Debug("Server.Watch defer called")
//		// FIXED: let sender closeFn channel instead of receiver.
//		// close(changeCh)
//		for _, ob := range obs {
//			s.watcher.Unsubscribe(ob)
//		}
//	}()
//
//	for {
//		// loop forever
//		select {
//		case changes := <-changeCh:
//			log.
//				WithField("changes", changes).
//				Warn("Server(grpc).Watch will be send to client")
//
//			if err = stream.Send(&pb.Changes{
//				Key: changes.Key,
//				Key:       changes.Key,
//				Format:    toPBFormat(changes.Format),
//				Checksum:  changes.CheckSum,
//				Data:      changes.Data,
//			}); err != nil {
//				log.Errorf("Server(grpc).Watch gets failed to send to client: %v", err)
//				continue
//			}
//
//		case <-stream.Context().Done():
//			// FIXED: what is the timing to quit and release resources timely.
//			log.Debug("Server(grpc).Watch received stream done signal, now quit")
//			return
//
//		case <-s.quit:
//			// if server quit, all watcher should quit too.
//			return
//
//		}
//	}
//}
//
//func New() *grpc.Server {
//	srv := &Server{
//		//addr: addr,
//		// 1 has no meaning, just want to call watcher.NewChannelWatcher.
//		watcher: watcher.NewChannelWatcher(1),
//		quit:    make(chan struct{}, 1),
//	}
//
//	// DONE(@yeqown): recover and logger interceptor needed
//	s := grpc.NewServer(
//		grpc.UnaryInterceptor(chainUnaryServer(serverRecovery(), serverLogger())),
//	)
//	pb.RegisterWatcherServer(s, srv)
//	reflection.Register(s)
//
//	return s
//}
//
//// ListenAndServe [DO NOT USE] only works for testing.
//func ListenAndServe(addr string) error {
//	srv := &Server{
//		//addr: addr,
//		// 1 has no meaning, just want to call watcher.NewChannelWatcher.
//		watcher: watcher.NewChannelWatcher(1),
//		quit:    make(chan struct{}, 1),
//	}
//
//	lis, err := net.Listen("tcp", addr)
//	if err != nil {
//		return err
//	}
//
//	s := grpc.NewServer()
//	pb.RegisterWatcherServer(s, srv)
//	reflection.Register(s)
//
//	return s.Serve(lis)
//}
