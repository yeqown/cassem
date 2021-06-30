package httpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
)

var (
	client = &http.Client{
		Timeout: 30 * time.Second,
	}
)

func POST(base string, body interface{}, resp interface{}) error {
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return errors.Wrap(err, "could not encode body")
	}

	req, err := http.NewRequest(http.MethodPost, base, buf)
	if err != nil {
		log.Errorf("invalid http.NewRequest: %v", err)
		return errors.Wrap(err, "invalid http.NewRequest")
	}
	req.Header.Set("Content-Type", "application/json")

	return execute(req, resp)
}

func GET(base string, data map[string]string, resp interface{}) error {
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

	return execute(req, resp)
}

func execute(req *http.Request, resp interface{}) error {
	r, err := client.Do(req)
	if err != nil {
		log.Errorf("invalid do: %v", err)
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		err = errors.New("response code:" + strconv.Itoa(r.StatusCode))
		if err2 := json.NewDecoder(r.Body).Decode(resp); err2 != nil {
			log.Errorf("execute could not parse response: %v", err2)
			return errors.Wrap(err, err2.Error())
		}
	}

	return err
}
