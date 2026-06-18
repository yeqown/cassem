package conf

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRaft_Fix(t *testing.T) {
	tests := []struct {
		name        string
		raft        *Raft
		expectError bool
		expectedID  uint64
	}{
		{
			name:        "empty bind",
			raft:        &Raft{},
			expectError: true,
		},
		{
			name: "empty cluster - defaults to bind",
			raft: &Raft{
				Bind:    "127.0.0.1:3021",
				Cluster: "",
			},
			expectError: false,
			expectedID:  1,
		},
		{
			name: "single node cluster",
			raft: &Raft{
				Bind:    "127.0.0.1:3021",
				Cluster: "127.0.0.1:3021",
			},
			expectError: false,
			expectedID:  1,
		},
		{
			name: "multi node cluster - first node",
			raft: &Raft{
				Bind:    "node1:3021",
				Cluster: "node1:3021,node2:3021,node3:3021",
			},
			expectError: false,
			expectedID:  1,
		},
		{
			name: "multi node cluster - second node",
			raft: &Raft{
				Bind:    "node2:3021",
				Cluster: "node1:3021,node2:3021,node3:3021",
			},
			expectError: false,
			expectedID:  2,
		},
		{
			name: "multi node cluster - third node",
			raft: &Raft{
				Bind:    "node3:3021",
				Cluster: "node1:3021,node2:3021,node3:3021",
			},
			expectError: false,
			expectedID:  3,
		},
		{
			name: "bind not in cluster",
			raft: &Raft{
				Bind:    "node4:3021",
				Cluster: "node1:3021,node2:3021,node3:3021",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.raft.Fix()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.raft.NodeID)
				if tt.raft.Cluster == "" {
					assert.Equal(t, tt.raft.Bind, tt.raft.Cluster)
				}
			}
		})
	}
}

func TestCassemdbListenAndAdvertiseAddrConfig(t *testing.T) {
	c := &CassemdbConfig{ListenAddr: "0.0.0.0:2021", AdvertiseAddr: "127.0.0.1:2021"}
	assert.Equal(t, "0.0.0.0:2021", c.ListenAddr)
	assert.Equal(t, "127.0.0.1:2021", c.AdvertiseAddr)
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func() string
		cleanup     func()
		target      any
		expectError bool
	}{
		{
			name: "valid TOML file",
			setupFile: func() string {
				content := `[server]
addr = "localhost:8080"

[raft]
bind = "127.0.0.1:3021"
cluster = "127.0.0.1:3021"
snapCount = 10000`
				f, _ := os.CreateTemp("", "test_*.toml")
				f.WriteString(content)
				f.Close()
				return f.Name()
			},
			cleanup: func() {
				// cleanup handled by temp file removal in setup
			},
			target: &struct {
				Server *Server `toml:"server"`
				Raft   *Raft   `toml:"raft"`
			}{},
			expectError: false,
		},
		{
			name: "file not found",
			setupFile: func() string {
				return "/nonexistent/path/config.toml"
			},
			cleanup:     func() {},
			target:      &struct{}{},
			expectError: true,
		},
		{
			name: "invalid TOML syntax",
			setupFile: func() string {
				content := `[server
addr = "localhost:8080"`
				f, _ := os.CreateTemp("", "test_*.toml")
				f.WriteString(content)
				f.Close()
				return f.Name()
			},
			cleanup:     func() {},
			target:      &struct{}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFile()
			if path != "" && !tt.expectError {
				defer os.Remove(path)
			}

			err := Load(path, tt.target)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify the config was loaded
				if cfg, ok := tt.target.(*struct {
					Server *Server `toml:"server"`
					Raft   *Raft   `toml:"raft"`
				}); ok {
					assert.NotNil(t, cfg.Server)
					assert.Equal(t, "localhost:8080", cfg.Server.Addr)
					assert.NotNil(t, cfg.Raft)
					assert.Equal(t, "127.0.0.1:3021", cfg.Raft.Bind)
				}
			}
		})
	}
}

func TestLoad_PanicOnEmptyPath(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			assert.Contains(t, r.(string), "todo load conf automatically")
		} else {
			t.Error("Expected panic on empty path")
		}
	}()

	Load("", &struct{}{})
}
