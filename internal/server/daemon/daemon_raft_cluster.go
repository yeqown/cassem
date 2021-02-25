package daemon

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var (
	client = &http.Client{}
)

type operateNodeResp struct {
	ErrCode    int         `json:"errcode"`
	ErrMessage string      `json:"errmsg,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

func operateNodeRequest(base string, data map[string]string) error {
	if base == "" {
		log.Warn("operateNodeRequest could not be executed with empty RAFT bind address, skip")
		return nil
	}
	// detection and fix schema
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}

	// assemble form parameters
	form := url.Values{}
	for k, v := range data {
		form.Add(k, v)
	}

	uri := base + "?" + form.Encode()
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		log.Errorf("invalid http.NewRequest: %v", err)
		return errors.Wrap(err, "invalid http.NewRequest")
	}

	r, err := client.Do(req)
	if err != nil {
		log.Errorf("invalid do: %v", err)
		return err
	}

	if r.StatusCode != http.StatusOK {
		defer r.Body.Close()
		result := new(operateNodeResp)
		if err = json.NewDecoder(r.Body).Decode(result); err != nil {
			log.Errorf("executeOperateNodeRequest could not parse response: %v", err)
			return err
		}

		err = errors.New(result.ErrMessage)
	}

	return err
}

const (
	_formServerId        = "serverId"
	_formAction          = "action"
	_formRaftBindAddress = "bind"

	_actionJoin = "join"
	_actionLeft = "left"
)

func (d *Daemon) tryJoinCluster() (err error) {
	base := d.cfg.Server.Raft.Join
	if err = operateNodeRequest(base, map[string]string{
		_formServerId:        d.serverId,
		_formAction:          _actionJoin,
		_formRaftBindAddress: d.cfg.Server.Raft.Bind,
	}); err != nil {
		log.Errorf("invalid request: %v", err)

		return errors.Wrap(err, "invalid http.NewRequest")
	}

	d.joinedCluster = true

	return
}

func (d *Daemon) tryLeaveCluster() (err error) {
	base := d.cfg.Server.Raft.Join
	if err = operateNodeRequest(base, map[string]string{
		_formServerId: d.serverId,
		_formAction:   _actionLeft,
	}); err != nil {
		log.Errorf("invalid request: %v", err)

		return errors.Wrap(err, "invalid http.NewRequest")
	}

	d.joinedCluster = false

	return
}
