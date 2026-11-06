package adm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"github.com/yeqown/cassem/internal/concept"
)

// Literal expectations preserve the HTTP contract formerly provided by the
// protobuf types' MarshalJSON methods, independent of the new adapters.
func TestElementHTTPPreservesProtobufJSON(t *testing.T) {
	tests := []struct {
		name    string
		element *concept.Element
		want    string
	}{
		{"nil", nil, `null`},
		{"zero", &concept.Element{}, `{"raw":"","version":0,"published":false}`},
		{"zero metadata", &concept.Element{Metadata: &concept.ElementMetadata{}}, `{"metadata":{"key":"","app":"","env":"","latestVersion":0,"unpublishedVersion":0,"usingVersion":0,"usingFingerprint":"","contentType":"UNKNOWN"},"raw":"","version":0,"published":false}`},
		{"known enum and bytes", &concept.Element{
			Metadata: &concept.ElementMetadata{Key: "config", App: "checkout", Env: "prod", LatestVersion: 9, UnpublishedVersion: 8, UsingVersion: 7, UsingFingerprint: "abc", ContentType: concept.ContentType_JSON},
			Raw:      []byte{0, 255, 128}, Version: 7, Published: true,
		}, `{"metadata":{"key":"config","app":"checkout","env":"prod","latestVersion":9,"unpublishedVersion":8,"usingVersion":7,"usingFingerprint":"abc","contentType":"JSON"},"raw":"AP+A","version":7,"published":true}`},
		{"unknown enum", &concept.Element{Metadata: &concept.ElementMetadata{ContentType: concept.ContentType(99)}}, `{"metadata":{"key":"","app":"","env":"","latestVersion":0,"unpublishedVersion":0,"usingVersion":0,"usingFingerprint":"","contentType":99},"raw":"","version":0,"published":false}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := app{aggregate: &elementJSONAggregate{element: tt.element}}
			router := chi.NewRouter()
			router.Get("/apps/{appId}/envs/{env}/elements/{key}", d.GetAppEnvElementHTTP)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/apps/checkout/envs/prod/elements/config", nil))
			require.Equal(t, http.StatusOK, resp.Code)
			require.JSONEq(t, `{"errcode":0,"errmsg":"success","data":`+tt.want+`}`, resp.Body.String())
		})
	}
}

func TestElementListsHTTPPreserveProtobufJSONAndPaging(t *testing.T) {
	tests := []struct {
		name   string
		result *concept.GetElementsResult
		want   string
	}{
		{"nil result", nil, `null`},
		{"nil list", &concept.GetElementsResult{}, `{"hasMore":false,"nextSeek":"","elements":null}`},
		{"empty list", &concept.GetElementsResult{Elements: []*concept.Element{}}, `{"hasMore":false,"nextSeek":"","elements":[]}`},
		{"nil item", &concept.GetElementsResult{Elements: []*concept.Element{nil}}, `{"hasMore":false,"nextSeek":"","elements":[null]}`},
		{"zero item and paging", &concept.GetElementsResult{
			CommonPager: concept.CommonPager{HasMore: true, NextSeek: "next/config"},
			Elements:    []*concept.Element{{}},
		}, `{"hasMore":true,"nextSeek":"next/config","elements":[{"raw":"","version":0,"published":false}]}`},
		{"metadata and bytes", &concept.GetElementsResult{Elements: []*concept.Element{{
			Metadata: &concept.ElementMetadata{ContentType: concept.ContentType_TOML}, Raw: []byte("a"),
		}, {Metadata: &concept.ElementMetadata{ContentType: concept.ContentType(99)}}}}, `{"hasMore":false,"nextSeek":"","elements":[{"metadata":{"key":"","app":"","env":"","latestVersion":0,"unpublishedVersion":0,"usingVersion":0,"usingFingerprint":"","contentType":"TOML"},"raw":"YQ==","version":0,"published":false},{"metadata":{"key":"","app":"","env":"","latestVersion":0,"unpublishedVersion":0,"usingVersion":0,"usingFingerprint":"","contentType":99},"raw":"","version":0,"published":false}]}`},
	}
	for _, route := range []struct {
		name string
		path string
	}{
		{"paged", "/apps/checkout/envs/prod/elements"},
		{"by keys", "/apps/checkout/envs/prod/elements?key=config"},
		{"versions", "/apps/checkout/envs/prod/elements/config/versions"},
	} {
		t.Run(route.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					agg := &elementJSONAggregate{result: tt.result}
					d := app{aggregate: agg}
					router := chi.NewRouter()
					router.Get("/apps/{appId}/envs/{env}/elements", d.GetAppEnvElementsHTTP)
					router.Get("/apps/{appId}/envs/{env}/elements/{key}/versions", d.GetAppEnvElementAllVersionsHTTP)
					resp := httptest.NewRecorder()
					router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, route.path, nil))
					require.Equal(t, http.StatusOK, resp.Code)
					require.JSONEq(t, `{"errcode":0,"errmsg":"success","data":`+tt.want+`}`, resp.Body.String())
					require.Equal(t, route.name, agg.called)
				})
			}
		})
	}
}

type elementJSONAggregate struct {
	concept.AdmAggregate
	element *concept.Element
	result  *concept.GetElementsResult
	called  string
}

func (f *elementJSONAggregate) GetElementWithVersion(context.Context, string, string, string, int) (*concept.Element, error) {
	return f.element, nil
}

func (f *elementJSONAggregate) GetElements(context.Context, string, string, string, int, string) (*concept.GetElementsResult, error) {
	f.called = "paged"
	return f.result, nil
}

func (f *elementJSONAggregate) GetElementsByKeys(context.Context, string, string, []string) (*concept.GetElementsResult, error) {
	f.called = "by keys"
	return f.result, nil
}

func (f *elementJSONAggregate) GetElementVersions(context.Context, string, string, string, string, int) (*concept.GetElementsResult, error) {
	f.called = "versions"
	return f.result, nil
}
