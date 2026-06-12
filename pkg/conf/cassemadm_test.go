package conf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCassemAdminConfig_Valid(t *testing.T) {
	tests := []struct {
		name        string
		config      *CassemAdminConfig
		expectError bool
		errMsg      string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errMsg:      "config is nil",
		},
		{
			name:        "empty endpoints",
			config:      &CassemAdminConfig{},
			expectError: true,
			errMsg:      "empty endpoints",
		},
		{
			name: "missing auth",
			config: &CassemAdminConfig{
				CassemDBEndpoints: []string{"127.0.0.1:2021"},
				HTTP:              &Server{Addr: ":8080"},
			},
			expectError: true,
			errMsg:      "empty auth session secret",
		},
		{
			name: "partial bootstrap admin",
			config: &CassemAdminConfig{
				CassemDBEndpoints: []string{"127.0.0.1:2021"},
				HTTP:              &Server{Addr: ":8080"},
				Auth: &AdminAuth{
					SessionSecret:    "test-session-secret",
					BootstrapAccount: "superadmin@example.com",
				},
			},
			expectError: true,
			errMsg:      "incomplete bootstrap admin",
		},
		{
			name: "valid config with one endpoint",
			config: &CassemAdminConfig{
				CassemDBEndpoints: []string{"127.0.0.1:2021"},
				HTTP:              &Server{Addr: ":8080"},
				Auth:              &AdminAuth{SessionSecret: "test-session-secret"},
			},
			expectError: false,
		},
		{
			name: "valid config with bootstrap admin",
			config: &CassemAdminConfig{
				CassemDBEndpoints: []string{
					"127.0.0.1:2021",
					"127.0.0.1:2022",
					"127.0.0.1:2023",
				},
				HTTP: &Server{Addr: ":8080"},
				Auth: &AdminAuth{
					SessionSecret:     "test-session-secret",
					BootstrapAccount:  "superadmin@example.com",
					BootstrapPassword: "password",
					BootstrapNickname: "superadmin",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Valid()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
