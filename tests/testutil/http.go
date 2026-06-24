package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/yeqown/cassem/api/concept"
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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

func LoginSuperadmin(t testing.TB, baseURL string) string {
	t.Helper()
	client := NewHTTPClient(baseURL)
	var resp struct {
		User    *concept.User `json:"user"`
		Session string        `json:"session"`
	}
	client.DoJSON(t, http.MethodPost, "/api/account/login", map[string]any{
		"account":  "superadmin@example.com",
		"password": "cassem",
	}, &resp)
	if resp.Session == "" {
		t.Fatal("empty superadmin session")
	}
	return resp.Session
}

// CheckCassemAdm verifies the admin control plane is ready by forcing a real login round-trip.
func CheckCassemAdm(baseURL string, account, password string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		err := checkCassemAdmOnce(baseURL, account, password, remaining)
		if err == nil {
			return nil
		}
		lastErr = err
		sleepUntilNextProbe(interval, deadline)
	}
	return fmt.Errorf("cassemadm %s did not become ready: %w", baseURL, lastErr)
}

func checkCassemAdmOnce(baseURL, account, password string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &http.Client{}
	body, err := json.Marshal(map[string]any{
		"account":  account,
		"password": password,
	})
	if err != nil {
		return fmt.Errorf("encode login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/account/login", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read login response: %w", err)
	}

	var out httpx.CommonResponse
	if err = json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("decode login response %q: %w", string(raw), err)
	}
	if resp.StatusCode/100 != 2 || out.ErrCode != httpx.OK {
		return fmt.Errorf("login returned status=%d errcode=%d errmsg=%s", resp.StatusCode, out.ErrCode, out.ErrMessage)
	}

	payload, err := json.Marshal(out.Data)
	if err != nil {
		return fmt.Errorf("encode login data: %w", err)
	}
	var data struct {
		Session string `json:"session"`
	}
	if err = json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("decode login data: %w", err)
	}
	if data.Session == "" {
		return fmt.Errorf("login response has empty session")
	}
	return nil
}
