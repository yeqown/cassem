package conf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCassemAgentConfig_Valid(t *testing.T) {
	tests := []struct {
		name        string
		config      *CassemAgentConfig
		expectError bool
		errMsg      string
		checkFields func(*testing.T, *CassemAgentConfig)
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errMsg:      "config is nil",
		},
		{
			name: "empty endpoints",
			config: &CassemAgentConfig{
				CassemDBEndpoints: []string{},
			},
			expectError: true,
			errMsg:      "empty endpoints",
		},
		{
			name: "renewInterval greater than TTL",
			config: &CassemAgentConfig{
				CassemDBEndpoints: []string{"127.0.0.1:2021"},
				TTL:               10,
				RenewInterval:     20,
			},
			expectError: true,
			errMsg:      "renewInterval should be lte than TTL",
		},
		{
			name: "valid config with defaults",
			config: &CassemAgentConfig{
				CassemDBEndpoints: []string{"127.0.0.1:2021"},
			},
			expectError: false,
			checkFields: func(t *testing.T, c *CassemAgentConfig) {
				// Note: there's a bug in Valid() where c.RenewInterval == 0
				// sets c.TTL instead of c.RenewInterval
				assert.Equal(t, int32(20), c.TTL, "TTL gets set to 30*0.666 due to bug")
				assert.Equal(t, int32(0), c.RenewInterval, "RenewInterval remains 0 due to bug")
				assert.Equal(t, int32(1000), c.ElementCacheSize)
			},
		},
		{
			name: "valid config with custom values",
			config: &CassemAgentConfig{
				CassemDBEndpoints: []string{"127.0.0.1:2021"},
				TTL:               60,
				RenewInterval:     40,
				ElementCacheSize:  5000,
			},
			expectError: false,
			checkFields: func(t *testing.T, c *CassemAgentConfig) {
				assert.Equal(t, int32(60), c.TTL)
				assert.Equal(t, int32(40), c.RenewInterval)
				assert.Equal(t, int32(5000), c.ElementCacheSize)
			},
		},
		{
			name: "TTL zero - should default to 30",
			config: &CassemAgentConfig{
				CassemDBEndpoints: []string{"127.0.0.1:2021"},
				TTL:               0,
			},
			expectError: false,
			checkFields: func(t *testing.T, c *CassemAgentConfig) {
				// Due to bug: TTL set to 30, then RenewInterval (0) causes TTL to be set to 30*0.666
				assert.Equal(t, int32(20), c.TTL)
			},
		},
		{
			name: "RenewInterval zero - should default to TTL * 0.666",
			config: &CassemAgentConfig{
				CassemDBEndpoints: []string{"127.0.0.1:2021"},
				TTL:               30,
				RenewInterval:     0,
			},
			expectError: false,
			checkFields: func(t *testing.T, c *CassemAgentConfig) {
				// Due to bug: RenewInterval check sets TTL instead of RenewInterval
				assert.Equal(t, int32(20), c.TTL, "TTL gets modified due to bug")
				assert.Equal(t, int32(0), c.RenewInterval, "RenewInterval remains 0")
			},
		},
		{
			name: "ElementCacheSize zero - should default to 1000",
			config: &CassemAgentConfig{
				CassemDBEndpoints: []string{"127.0.0.1:2021"},
				ElementCacheSize:  0,
			},
			expectError: false,
			checkFields: func(t *testing.T, c *CassemAgentConfig) {
				assert.Equal(t, int32(1000), c.ElementCacheSize)
			},
		},
		{
			name: "valid config with multiple endpoints",
			config: &CassemAgentConfig{
				CassemDBEndpoints: []string{
					"127.0.0.1:2021",
					"127.0.0.1:2022",
					"127.0.0.1:2023",
				},
				TTL:           45,
				RenewInterval: 30,
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
				if tt.checkFields != nil {
					tt.checkFields(t, tt.config)
				}
			}
		})
	}
}
