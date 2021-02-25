package daemon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var (
	client = &http.Client{}
)

func sendReq(req *http.Request) error {
	r, err := client.Do(req)
	if err != nil {
		log.Errorf("invalid do: %v", err)
		return err
	}
	defer r.Body.Close()
	byts, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("invalid do: %v", err)
		return err
	}

	resp := new(commonResp)
	if err = json.Unmarshal(byts, resp); err != nil {
		return err
	}

	if resp.Code != 0 {
		err = errors.New("request failed: " + resp.Errmsg)
	}

	return err
}

func (d *Daemon) join() error {
	form := url.Values{}
	form.Add("serverId", d.serverId)
	form.Add("action", "join")
	form.Add("addr", d.cfg.Server.Raft.Bind)

	uri := "http://" + d.cfg.Server.Raft.Join + "?" + form.Encode()
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		log.Errorf("invalid http.NewRequest: %v", err)
		return errors.Wrap(err, "invalid http.NewRequest")
	}

	if err = sendReq(req); err != nil {
		return errors.Wrapf(err, "sendReq failed")
	}

	d.joinedCluster = true
	return nil
}

func (d *Daemon) leave() error {
	form := url.Values{}
	form.Add("serverId", d.serverId)
	form.Add("action", "leave")

	uri := "http://" + d.cfg.Server.Raft.Join + "?" + form.Encode()
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		log.Errorf("invalid request: %v", err)
		return errors.Wrap(err, "invalid http.NewRequest")
	}

	if err = sendReq(req); err != nil {
		return errors.Wrapf(err, "sendReq failed")
	}

	d.joinedCluster = false
	return nil
}

func (d Daemon) serveClusterNode() error {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("cluster handler received a request")

		_ = req.ParseForm()
		serverId := req.Form.Get("serverId")
		addr := req.Form.Get("addr")
		action := req.Form.Get("action")

		var err error
		switch action {
		case "join":
			err = d.addNode(serverId, addr)
		case "leave":
			err = d.removeNode(serverId)
		default:
			err = errors.New("Unknown action")
		}

		if err != nil {
			responseError(w, err)
			return
		}

		responseOK(w)
	})

	return http.ListenAndServe(d.cfg.Server.Raft.Listen, nil)
}

type commonResp struct {
	Code   int    `json:"result"`
	Errmsg string `json:"errmsg"`
}

func responseError(w http.ResponseWriter, err error) {
	r := commonResp{
		Code:   -1,
		Errmsg: err.Error(),
	}
	byts, _ := json.Marshal(r)
	_, _ = fmt.Fprint(w, string(byts))
}

func responseOK(w http.ResponseWriter) {
	r := commonResp{
		Code:   0,
		Errmsg: "success",
	}
	byts, _ := json.Marshal(r)
	_, _ = fmt.Fprint(w, string(byts))
}
