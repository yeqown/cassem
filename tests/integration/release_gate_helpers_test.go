//go:build integration

package integration_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/tests/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type releaseGateAdmClient struct {
	*testutil.HTTPClient
}

func newReleaseGateAdm(t testing.TB, baseURL string) releaseGateAdmClient {
	t.Helper()

	client := testutil.NewHTTPClient(baseURL)
	client.Session = testutil.LoginSuperadmin(t, baseURL)
	return releaseGateAdmClient{HTTPClient: client}
}

func createAppEnv(t testing.TB, adm releaseGateAdmClient, app string, env string) {
	t.Helper()

	adm.DoJSON(t, http.MethodPost, "/api/apps/"+app, map[string]any{
		"name":        app,
		"description": "release gate app " + app,
	}, nil)
	adm.DoJSON(t, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s", app, env), nil, nil)
}

func createJSONElement(t testing.TB, adm releaseGateAdmClient, app string, env string, key string, raw string) {
	t.Helper()

	adm.DoJSON(t, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{
		"raw":         raw,
		"contentType": concept.ContentType_JSON,
	}, nil)
}

func updateElementRaw(t testing.TB, adm releaseGateAdmClient, app string, env string, key string, raw string) {
	t.Helper()

	adm.DoJSON(t, http.MethodPut, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key), map[string]any{
		"raw": raw,
	}, nil)
}

func publishReleaseGateElement(t testing.TB, adm releaseGateAdmClient, app string, env string, key string, version int32, mode concept.PublishingMode, instanceIDs []string) {
	t.Helper()

	v := url.Values{}
	v.Set("version", fmt.Sprintf("%d", version))
	v.Set("publishMode", fmt.Sprintf("%d", mode))
	for _, id := range instanceIDs {
		v.Add("instanceId", id)
	}
	adm.DoJSON(t, http.MethodPost, fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s/publish?%s", app, env, key, v.Encode()), nil, nil)
}

func getElement(t testing.TB, adm releaseGateAdmClient, app string, env string, key string, version int32) *concept.Element {
	t.Helper()

	path := fmt.Sprintf("/api/apps/%s/envs/%s/elements/%s", app, env, key)
	if version > 0 {
		path += fmt.Sprintf("?version=%d", version)
	}

	var resp struct {
		Metadata  elementMetadataView `json:"metadata"`
		Raw       string              `json:"raw"`
		Version   int32               `json:"version"`
		Published bool                `json:"published"`
	}
	adm.DoJSON(t, http.MethodGet, path, nil, &resp)
	raw, err := base64.StdEncoding.DecodeString(resp.Raw)
	require.NoError(t, err)
	return &concept.Element{
		Metadata:  resp.Metadata.toProto(),
		Raw:       raw,
		Version:   resp.Version,
		Published: resp.Published,
	}
}

type elementMetadataView struct {
	App                string `json:"app"`
	Env                string `json:"env"`
	Key                string `json:"key"`
	ContentType        string `json:"contentType"`
	LatestVersion      int32  `json:"latestVersion"`
	UsingVersion       int32  `json:"usingVersion"`
	UnpublishedVersion int32  `json:"unpublishedVersion"`
	UsingFingerprint   string `json:"usingFingerprint"`
}

func (v elementMetadataView) toProto() *concept.ElementMetadata {
	return &concept.ElementMetadata{
		App:                v.App,
		Env:                v.Env,
		Key:                v.Key,
		ContentType:        contentTypeFromString(v.ContentType),
		LatestVersion:      v.LatestVersion,
		UsingVersion:       v.UsingVersion,
		UnpublishedVersion: v.UnpublishedVersion,
		UsingFingerprint:   v.UsingFingerprint,
	}
}

func contentTypeFromString(value string) concept.ContentType {
	switch value {
	case "JSON":
		return concept.ContentType_JSON
	case "TOML":
		return concept.ContentType_TOML
	case "INI":
		return concept.ContentType_INI
	case "PLAINTEXT":
		return concept.ContentType_PLAINTEXT
	default:
		return concept.ContentType_UNKNOWN
	}
}

type agentElementGetter interface {
	GetElement(ctx context.Context, app string, env string, keys ...string) ([]*concept.Element, error)
}

func waitAgentRaw(t testing.TB, client agentElementGetter, app string, env string, key string, expected string) {
	t.Helper()

	waitAgentRawWithin(t, client, app, env, key, expected, 10*time.Second)
}

func waitAgentRawWithin(t testing.TB, client agentElementGetter, app string, env string, key string, expected string, timeout time.Duration) {
	t.Helper()

	testutil.RequireEventually(t, timeout, 200*time.Millisecond, func() (bool, string) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		elems, err := client.GetElement(ctx, app, env, key)
		if err != nil {
			return false, err.Error()
		}
		if len(elems) != 1 {
			return false, fmt.Sprintf("got %d elements", len(elems))
		}
		if elems[0] == nil {
			return false, "got nil element"
		}
		if string(elems[0].GetRaw()) != expected {
			return false, fmt.Sprintf("raw=%q", string(elems[0].GetRaw()))
		}
		return true, ""
	})
}

func waitWatchRegistered(t testing.TB, endpoint string, app string, env string, key string, instanceID string, received <-chan *concept.Element) {
	t.Helper()

	cc, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer cc.Close()

	client := agent.NewDeliveryClient(cc)
	probe := &concept.Element{Metadata: &concept.ElementMetadata{App: app, Env: env, Key: key}, Raw: []byte("probe")}
	testutil.RequireEventually(t, 10*time.Second, 200*time.Millisecond, func() (bool, string) {
		callCtx, callCancel := context.WithTimeout(context.Background(), time.Second)
		_, err := client.Dispatch(callCtx, &agent.DispatchReq{Elems: []*concept.Element{probe}, InstanceIds: []string{instanceID}})
		callCancel()
		if err != nil {
			return false, err.Error()
		}
		select {
		case got := <-received:
			if got == nil {
				return false, "received nil element"
			}
			if string(got.GetRaw()) == "probe" {
				return true, ""
			}
			return false, fmt.Sprintf("unexpected raw=%q", string(got.GetRaw()))
		case <-time.After(100 * time.Millisecond):
			return false, "probe not delivered"
		}
	})

	for {
		select {
		case <-received:
		default:
			return
		}
	}
}

func waitForRaw(t testing.TB, received <-chan *concept.Element, expected string, timeout time.Duration) {
	t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case got := <-received:
			if got != nil && string(got.GetRaw()) == expected {
				return
			}
		case <-deadline:
			t.Fatalf("did not receive expected element raw=%q", expected)
		}
	}
}

func assertNoRaw(t testing.TB, received <-chan *concept.Element, unexpected string, timeout time.Duration) {
	t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case got := <-received:
			if got != nil && string(got.GetRaw()) == unexpected {
				t.Fatalf("received unexpected element raw=%q", unexpected)
			}
		case <-deadline:
			return
		}
	}
}
