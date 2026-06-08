package testutil

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/yeqown/cassem/pkg/httpx"
)

type HTTPClient struct {
	BaseURL string
	Session string
	Client  *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *HTTPClient) DoJSON(t testing.TB, method string, path string, body any, data any) {
	t.Helper()

	var r io.Reader
	if body != nil {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		r = buf
	}

	req, err := http.NewRequest(method, c.BaseURL+path, r)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Session != "" {
		req.Header.Set("x-cassem-session", c.Session)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		t.Fatalf("do %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("%s %s returned %d: %s", method, path, resp.StatusCode, string(raw))
	}

	var out httpx.CommonResponse
	if err = json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode common response %q: %v", string(raw), err)
	}
	if out.ErrCode != httpx.OK {
		t.Fatalf("%s %s returned errcode %d: %s", method, path, out.ErrCode, out.ErrMessage)
	}

	if data == nil || out.Data == nil {
		return
	}
	payload, err := json.Marshal(out.Data)
	if err != nil {
		t.Fatalf("re-marshal response data: %v", err)
	}
	if err = json.Unmarshal(payload, data); err != nil {
		t.Fatalf("decode response data into %T: %v; payload=%s", data, err, string(payload))
	}
}

func (c *HTTPClient) DoExpectError(t testing.TB, method string, path string, body any) {
	t.Helper()

	var r io.Reader
	if body != nil {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		r = buf
	}

	req, err := http.NewRequest(method, c.BaseURL+path, r)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Session != "" {
		req.Header.Set("x-cassem-session", c.Session)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		t.Fatalf("do %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var out httpx.CommonResponse
		if err = json.Unmarshal(raw, &out); err == nil && out.ErrCode == httpx.OK {
			t.Fatalf("%s %s unexpectedly succeeded: %s", method, path, string(raw))
		}
	}
}

func SuperadminSession() string {
	val, err := json.Marshal(struct {
		Account   string
		Salt      string
		ExpiredAt int64
	}{
		Account:   "superadmin",
		Salt:      "Y2Fzc2VuCg==",
		ExpiredAt: time.Now().Add(24 * time.Hour).Unix(),
	})
	if err != nil {
		panic(fmt.Errorf("marshal superadmin session: %w", err))
	}

	return base64.StdEncoding.EncodeToString(val)
}
