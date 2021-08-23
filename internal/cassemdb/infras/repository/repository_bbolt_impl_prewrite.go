package repository

//
//import (
//	"fmt"
//	"time"
//
//	bolt "go.etcd.io/bbolt"
//)
//
//// preWriteLog help boltRepoImpl to speed up write operations.
//type preWriteLog struct {
//	// id is the unique sequence number for the log in one batch.
//	id int
//
//	// execFn contains all operations to achieve.
//	execFn func(tx *bolt.Tx) error
//
//	resp *preWriteResp
//}
//
//type preWriteResp struct {
//	id int
//
//	// err has 1 buffer, so it's ok to dispatch to multi-requests.
//	err chan error
//}
//
//func (p *preWriteResp) Err() <-chan error {
//	return p.err
//}
//
//// preWrite execute write operation into one channel.
//// make sure has pre-checked the errors those would encounter during the exec procedure.
//func (b boltRepoImpl) preWrite(pw *preWriteLog) *preWriteResp {
//	resp := &preWriteResp{
//		id:  0,
//		err: make(chan error, 1),
//	}
//	pw.resp = resp
//
//	select {
//	case b.preWriteC <- pw:
//	}
//
//	return resp
//}
//
//const (
//	_BATCH_SIZE                 = 100
//	_PRE_WRITE_BUF_SIZE         = 512
//	_PRE_WRITE_WAIT_PERIOD      = 10  // milliseconds
//	_PRE_WRITE_MAX_WRIAT_PERIOD = 100 // milliseconds
//)
//
//var (
//	_MaxWaitTimes = _PRE_WRITE_MAX_WRIAT_PERIOD / _PRE_WRITE_WAIT_PERIOD
//)
//
//// preWriteDispatcher will process preWriteLog in order and return result to client,
//// it works in a standalone goroutine, and make the response to be synchronized by channel.
//func (b boltRepoImpl) preWriteDispatcher() error {
//	var (
//		batch     = make([]*preWriteLog, 0, _BATCH_SIZE+5)
//		waitTimes int
//		ticker    = time.NewTicker(_PRE_WRITE_WAIT_PERIOD * time.Millisecond)
//	)
//
//	// reset all count
//	reset := func() {
//		// batch add some buffer, avoid overflow which cause slice memory allocation.
//		// is it needed?
//		batch = make([]*preWriteLog, 0, _BATCH_SIZE+5)
//		waitTimes = 0
//		ticker.Reset(_PRE_WRITE_WAIT_PERIOD * time.Millisecond)
//	}
//
//	for {
//		select {
//		case pw := <-b.preWriteC:
//			// set preWriteLog's id.
//			pw.id = len(batch)
//			batch = append(batch, pw)
//
//			// if overflow the maximum batch size.
//			if len(batch) >= _BATCH_SIZE {
//				b.execPreWriteBatch(batch)
//				reset()
//				continue
//			}
//
//			// overflow than max wait time (100ms)
//			if waitTimes++; waitTimes >= _MaxWaitTimes {
//				b.execPreWriteBatch(batch)
//				reset()
//				continue
//			}
//			// otherwise, wait more _PRE_WRITE_WAIT_PERIOD(10ms)
//			ticker.Reset(_PRE_WRITE_WAIT_PERIOD * time.Millisecond)
//
//		case <-ticker.C:
//			// ticker reached and no new pre-write log anymore, just write.
//			b.execPreWriteBatch(batch)
//			reset()
//			continue
//		}
//	}
//}
//
//func (b boltRepoImpl) execPreWriteBatch(batch []*preWriteLog) {
//	if len(batch) == 0 {
//		// empty batch means no need to execute.
//		return
//	}
//
//	errs := make([]error, 0, len(batch))
//	err := b.db.Batch(func(tx *bolt.Tx) error {
//		for _, pw := range batch {
//			err := pw.execFn(tx)
//			errs = append(errs, err)
//		}
//
//		return nil
//	})
//
//	if err != nil {
//		b.dispatchResult(batch, nil, err)
//		return
//	}
//
//	b.dispatchResult(batch, errs, nil)
//	return
//}
//
//func (b boltRepoImpl) dispatchResult(batch []*preWriteLog, respErrs []error, err error) {
//	if len(respErrs) != len(batch) {
//		panic(fmt.Sprintf("mismatched size batch(%d) and respErrs(%d)",
//			len(batch), len(respErrs)))
//	}
//
//	// common error to all batch resp.
//	if err != nil {
//		for _, pw := range batch {
//			pw.resp.err <- err
//			close(pw.resp.err)
//		}
//		return
//	}
//
//	for idx, pw := range batch {
//		pw.resp.err <- respErrs[idx]
//		close(pw.resp.err)
//	}
//}
