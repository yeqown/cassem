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
		check       func(*testing.T, *CassemAdminConfig)
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
			name: "valid config with one endpoint applies retention defaults",
			config: &CassemAdminConfig{
				CassemDBEndpoints: []string{"127.0.0.1:2021"},
				HTTP:              &Server{Addr: ":8080"},
				Auth:              &AdminAuth{SessionSecret: "test-session-secret"},
			},
			expectError: false,
			check: func(t *testing.T, cfg *CassemAdminConfig) {
				if assert.NotNil(t, cfg.Retention) {
					assert.True(t, cfg.Retention.EnabledValue())
					assert.Equal(t, "10m", cfg.Retention.Interval)
					assert.Equal(t, 20, cfg.Retention.MaxElementsPerRunValue())
					assert.Equal(t, 100, cfg.Retention.ElementPageSizeValue())
					assert.Equal(t, 20, cfg.Retention.KeepVersionCountValue())
					assert.Equal(t, 30, cfg.Retention.KeepVersionDaysValue())
					assert.Equal(t, 180, cfg.Retention.KeepOperationDaysValue())
					assert.Equal(t, "48h", cfg.Retention.FailureResultTTL)
				}
			},
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
				return
			}

			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, tt.config)
			}
		})
	}
}

func TestRetentionConfig(t *testing.T) {
	t.Run("invalid interval rejected", func(t *testing.T) {
		cfg := &RetentionConfig{
			Interval:          "bad",
			MaxElementsPerRun: new(20),
			ElementPageSize:   new(100),
			KeepVersionCount:  new(20),
			KeepVersionDays:   new(30),
			KeepOperationDays: new(180),
			FailureResultTTL:  "48h",
		}

		err := cfg.Fix()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid retention interval")
	})

	t.Run("disabled config accepted", func(t *testing.T) {
		enabled := false
		cfg := &RetentionConfig{Enabled: &enabled}

		err := cfg.Fix()

		assert.NoError(t, err)
		assert.False(t, cfg.EnabledValue())
		assert.Equal(t, "10m", cfg.Interval)
		assert.Equal(t, 20, cfg.MaxElementsPerRunValue())
		assert.Equal(t, 100, cfg.ElementPageSizeValue())
		assert.Equal(t, 20, cfg.KeepVersionCountValue())
		assert.Equal(t, 30, cfg.KeepVersionDaysValue())
		assert.Equal(t, 180, cfg.KeepOperationDaysValue())
		assert.Equal(t, "48h", cfg.FailureResultTTL)
	})

	t.Run("invalid element page size rejected", func(t *testing.T) {
		cfg := &RetentionConfig{
			Interval:          "10m",
			MaxElementsPerRun: new(20),
			ElementPageSize:   new(101),
			KeepVersionCount:  new(20),
			KeepVersionDays:   new(30),
			KeepOperationDays: new(180),
			FailureResultTTL:  "48h",
		}

		err := cfg.Fix()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retention elementPageSize must be in [1, 100]")
	})

	t.Run("invalid durations rejected", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*RetentionConfig)
			errMsg string
		}{
			{
				name: "zero interval",
				mutate: func(cfg *RetentionConfig) {
					cfg.Interval = "0s"
				},
				errMsg: "retention interval must be positive",
			},
			{
				name: "negative interval",
				mutate: func(cfg *RetentionConfig) {
					cfg.Interval = "-1s"
				},
				errMsg: "retention interval must be positive",
			},
			{
				name: "zero failure ttl",
				mutate: func(cfg *RetentionConfig) {
					cfg.FailureResultTTL = "0s"
				},
				errMsg: "retention failureResultTTL must be positive",
			},
			{
				name: "negative failure ttl",
				mutate: func(cfg *RetentionConfig) {
					cfg.FailureResultTTL = "-1s"
				},
				errMsg: "retention failureResultTTL must be positive",
			},
			{
				name: "too large failure ttl",
				mutate: func(cfg *RetentionConfig) {
					cfg.FailureResultTTL = "49h"
				},
				errMsg: "retention failureResultTTL must be less than or equal to 48h0m0s",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := DefaultRetentionConfig()
				tt.mutate(cfg)

				err := cfg.Fix()

				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			})
		}
	})

	t.Run("explicit zero rejected for positive numeric fields", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*RetentionConfig)
			errMsg string
		}{
			{
				name: "max elements per run",
				mutate: func(cfg *RetentionConfig) {
					cfg.MaxElementsPerRun = new(0)
				},
				errMsg: "retention maxElementsPerRun must be positive",
			},
			{
				name: "element page size",
				mutate: func(cfg *RetentionConfig) {
					cfg.ElementPageSize = new(0)
				},
				errMsg: "retention elementPageSize must be in [1, 100]",
			},
			{
				name: "keep version count",
				mutate: func(cfg *RetentionConfig) {
					cfg.KeepVersionCount = new(0)
				},
				errMsg: "retention keepVersionCount must be positive",
			},
			{
				name: "keep version days",
				mutate: func(cfg *RetentionConfig) {
					cfg.KeepVersionDays = new(0)
				},
				errMsg: "retention keepVersionDays must be positive",
			},
			{
				name: "keep operation days",
				mutate: func(cfg *RetentionConfig) {
					cfg.KeepOperationDays = new(0)
				},
				errMsg: "retention keepOperationDays must be positive",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := DefaultRetentionConfig()
				tt.mutate(cfg)

				err := cfg.Fix()

				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			})
		}
	})

	t.Run("upper bounds rejected for retention numeric fields", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(*RetentionConfig)
			errMsg string
		}{
			{
				name: "keep version count",
				mutate: func(cfg *RetentionConfig) {
					cfg.KeepVersionCount = new(maxRetentionKeepVersionCount + 1)
				},
				errMsg: "retention keepVersionCount must be in [1, 10000]",
			},
			{
				name: "keep version days",
				mutate: func(cfg *RetentionConfig) {
					cfg.KeepVersionDays = new(maxRetentionKeepVersionDays + 1)
				},
				errMsg: "retention keepVersionDays must be in [1, 36500]",
			},
			{
				name: "keep operation days",
				mutate: func(cfg *RetentionConfig) {
					cfg.KeepOperationDays = new(maxRetentionKeepOperationDays + 1)
				},
				errMsg: "retention keepOperationDays must be in [1, 36500]",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := DefaultRetentionConfig()
				tt.mutate(cfg)

				err := cfg.Fix()

				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			})
		}
	})

	t.Run("defaults only fill omitted fields", func(t *testing.T) {
		cfg := &RetentionConfig{}

		err := cfg.Fix()

		assert.NoError(t, err)
		assert.Equal(t, 20, cfg.MaxElementsPerRunValue())
		assert.Equal(t, 100, cfg.ElementPageSizeValue())
		assert.Equal(t, 20, cfg.KeepVersionCountValue())
		assert.Equal(t, 30, cfg.KeepVersionDaysValue())
		assert.Equal(t, 180, cfg.KeepOperationDaysValue())
		if assert.NotNil(t, cfg.MaxElementsPerRun) {
			assert.Equal(t, 20, *cfg.MaxElementsPerRun)
		}
	})
}
