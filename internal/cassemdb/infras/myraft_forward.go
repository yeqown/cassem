package infras

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yeqown/cassem/pkg/httpx"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

type forwardRequest struct {
	forceBase string
	path      string
	method    string
	form      map[string]string
	body      interface{}
}

//// operateNodeResp is a copy from internal/api/http.commonResponse, only be used to
//// be unmarshalled from response of myraft.tryJoinCluster.
//type operateNodeResp struct {
//	ErrCode    int    `json:"errcode"`
//	ErrMessage string `json:"errmsg,omitempty"`
//}

var (
	getRestyOnce = sync.Once{}
	_resty       *resty.Client
)

func getResty() *resty.Client {
	getRestyOnce.Do(func() {
		_resty = resty.New().
			SetTimeout(10 * time.Second)
	})

	return _resty
}

// forwardToLeader only forward operations in core (apply, join, leave).
// this would send a request(HTTP) to leader contains what operation need to do, of course, it takes
// necessary external information.
//
// Only slaves should call this.
func (r *myraft) forwardToLeader(req *forwardRequest) (err error) {
	base := r.fsm.getLeaderAddr()
	if req.forceBase != "" {
		base = req.forceBase
	}

	// detection base empty or not, fix schema and assemble path to base
	if base == "" {
		log.Warn("forwardToLeader could not be executed with empty RAFT bind address, skip")
		return nil
	}

	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	if strings.HasSuffix(base, "/") {
		base = strings.TrimRight(base, "/")
	}
	base += req.path

	resp := new(httpx.CommonResponse)
	rr := getResty().R().SetQueryParam("clusterSecret", "9520059dd167").SetResult(resp)

	switch req.method {
	case http.MethodGet:
		_, err = rr.SetQueryParams(req.form).Get(base)
	case http.MethodPost:
		_, err = rr.SetBody(req.body).Post(base)
	}

	if resp.ErrCode != 0 {
		err = errors.New(resp.ErrMessage)
	}

	return
}

func (r myraft) forwardToLeaderJoinLeft(action string, forceBase string) (err error) {
	form := map[string]string{
		_formServerId: r.serverId,
		_formAction:   action,
	}

	switch action {
	case _actionJoin:
		form[_formRaftBindAddress] = r.conf.Raft.RaftBind
	case _actionLeft:
	}

	req := forwardRequest{
		forceBase: forceBase,
		path:      "/cluster/nodes",
		method:    http.MethodGet,
		form:      form,
		body:      nil,
	}

	// DONE(@yeqown): should send request to leader
	if err = r.forwardToLeader(&req); err != nil {
		log.
			Errorf("myraft.forwardToLeaderJoinLeft calling r.forwardToLeader failed: %v", err)

		return errors.Wrap(err, "myraft.forwardToLeaderJoinLeft failed")
	}

	return nil
}

func (r myraft) forwardToLeaderApply(fsmLog *fsmLog) error {
	data, err := fsmLog.Serialize()
	if err != nil {
		return errors.Wrap(err, "myraft.forwardToLeaderApply failed to fsmLog.serialize")
	}

	req := forwardRequest{
		path:   "/cluster/apply",
		method: http.MethodPost,
		body: struct {
			ApplyData []byte `json:"Data"`
		}{
			ApplyData: data,
		},
	}

	if err = r.forwardToLeader(&req); err != nil {
		log.
			Errorf("myraft.forwardToLeaderApply calling r.forwardToLeader failed: %v", err)

		return errors.Wrap(err, "myraft.forwardToLeaderApply failed")
	}

	return nil
}
