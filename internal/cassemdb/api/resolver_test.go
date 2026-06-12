package api

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

type captureClientConn struct {
	state resolver.State
}

func (c *captureClientConn) UpdateState(state resolver.State) error {
	c.state = state
	return nil
}
func (c *captureClientConn) ReportError(error)                                    {}
func (c *captureClientConn) NewAddress([]resolver.Address)                        {}
func (c *captureClientConn) NewServiceConfig(string)                              {}
func (c *captureClientConn) ParseServiceConfig(string) *serviceconfig.ParseResult { return nil }

func TestCassemdbResolverBuildParsesEndpoint(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "write endpoints",
			path: "/cassemdb1:2021,cassemdb2:2021",
			want: []string{"cassemdb1:2021", "cassemdb2:2021"},
		},
		{
			name: "read endpoints with all prefix",
			path: "/all//cassemdb1:2021,cassemdb2:2021",
			want: []string{"cassemdb1:2021", "cassemdb2:2021"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &captureClientConn{}
			_, err := cassemdbResolverBuilder{}.Build(
				resolver.Target{URL: url.URL{Scheme: "cassemdb", Path: tt.path}},
				cc,
				resolver.BuildOptions{},
			)
			require.NoError(t, err)
			require.Len(t, cc.state.Addresses, len(tt.want))
			for i, want := range tt.want {
				require.Equal(t, want, cc.state.Addresses[i].Addr)
			}
		})
	}
}
