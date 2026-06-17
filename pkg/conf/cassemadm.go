package conf

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// CassemAdminConfig contains all config to cassemadm.
type CassemAdminConfig struct {
	// CassemDBEndpoints in format like: 127.0.0.1:2021 172.16.2.26:2021 172.16.2.27:2021
	CassemDBEndpoints []string `toml:"cassemdb"`

	// HTTP indicates which port is cassemadm serving on.
	HTTP *Server `toml:"http"`

	Auth      *AdminAuth       `toml:"auth"`
	Retention *RetentionConfig `toml:"retention"`
}

type RetentionConfig struct {
	Enabled           *bool  `toml:"enabled"`
	Interval          string `toml:"interval"`
	MaxElementsPerRun *int   `toml:"maxElementsPerRun"`
	ElementPageSize   *int   `toml:"elementPageSize"`
	KeepVersionCount  *int   `toml:"keepVersionCount"`
	KeepVersionDays   *int   `toml:"keepVersionDays"`
	KeepOperationDays *int   `toml:"keepOperationDays"`
	FailureResultTTL  string `toml:"failureResultTTL"`
}

const (
	defaultRetentionInterval          = "10m"
	defaultRetentionMaxElementsPerRun = 20
	defaultRetentionElementPageSize   = 100
	defaultRetentionKeepVersionCount  = 20
	defaultRetentionKeepVersionDays   = 30
	defaultRetentionKeepOperationDays = 180
	defaultRetentionFailureResultTTL  = "48h"

	maxRetentionKeepVersionCount  = 10000
	maxRetentionKeepVersionDays   = 36500
	maxRetentionKeepOperationDays = 36500
	maxRetentionFailureResultTTL  = 48 * time.Hour
)

func intValue(ptr *int, fallback int) int {
	if ptr == nil {
		return fallback
	}
	return *ptr
}

func DefaultRetentionConfig() *RetentionConfig {
	return &RetentionConfig{
		Enabled:           new(true),
		Interval:          defaultRetentionInterval,
		MaxElementsPerRun: new(defaultRetentionMaxElementsPerRun),
		ElementPageSize:   new(defaultRetentionElementPageSize),
		KeepVersionCount:  new(defaultRetentionKeepVersionCount),
		KeepVersionDays:   new(defaultRetentionKeepVersionDays),
		KeepOperationDays: new(defaultRetentionKeepOperationDays),
		FailureResultTTL:  defaultRetentionFailureResultTTL,
	}
}

func (r *RetentionConfig) EnabledValue() bool {
	if r == nil {
		return false
	}
	if r.Enabled == nil {
		return true
	}
	return *r.Enabled
}

func (r *RetentionConfig) MaxElementsPerRunValue() int {
	if r == nil {
		return 0
	}
	return intValue(r.MaxElementsPerRun, defaultRetentionMaxElementsPerRun)
}

func (r *RetentionConfig) ElementPageSizeValue() int {
	if r == nil {
		return 0
	}
	return intValue(r.ElementPageSize, defaultRetentionElementPageSize)
}

func (r *RetentionConfig) KeepVersionCountValue() int {
	if r == nil {
		return 0
	}
	return intValue(r.KeepVersionCount, defaultRetentionKeepVersionCount)
}

func (r *RetentionConfig) KeepVersionDaysValue() int {
	if r == nil {
		return 0
	}
	return intValue(r.KeepVersionDays, defaultRetentionKeepVersionDays)
}

func (r *RetentionConfig) KeepOperationDaysValue() int {
	if r == nil {
		return 0
	}
	return intValue(r.KeepOperationDays, defaultRetentionKeepOperationDays)
}

func (r *RetentionConfig) IntervalDuration() (time.Duration, error) {
	if r == nil {
		return 0, errors.New("retention config is nil")
	}
	return time.ParseDuration(r.Interval)
}

func (r *RetentionConfig) FailureResultTTLDuration() (time.Duration, error) {
	if r == nil {
		return 0, errors.New("retention config is nil")
	}
	return time.ParseDuration(r.FailureResultTTL)
}

func (r *RetentionConfig) Fix() error {
	if r == nil {
		return errors.New("retention config is nil")
	}

	defaults := DefaultRetentionConfig()
	if r.Enabled == nil {
		r.Enabled = new(defaults.EnabledValue())
	}
	if strings.TrimSpace(r.Interval) == "" {
		r.Interval = defaults.Interval
	}
	if r.MaxElementsPerRun == nil {
		r.MaxElementsPerRun = new(defaults.MaxElementsPerRunValue())
	}
	if r.ElementPageSize == nil {
		r.ElementPageSize = new(defaults.ElementPageSizeValue())
	}
	if r.KeepVersionCount == nil {
		r.KeepVersionCount = new(defaults.KeepVersionCountValue())
	}
	if r.KeepVersionDays == nil {
		r.KeepVersionDays = new(defaults.KeepVersionDaysValue())
	}
	if r.KeepOperationDays == nil {
		r.KeepOperationDays = new(defaults.KeepOperationDaysValue())
	}
	if strings.TrimSpace(r.FailureResultTTL) == "" {
		r.FailureResultTTL = defaults.FailureResultTTL
	}

	if interval, err := r.IntervalDuration(); err != nil {
		return fmt.Errorf("invalid retention interval: %w", err)
	} else if interval <= 0 {
		return errors.New("retention interval must be positive")
	}
	if failureResultTTL, err := r.FailureResultTTLDuration(); err != nil {
		return fmt.Errorf("invalid retention failureResultTTL: %w", err)
	} else if failureResultTTL <= 0 {
		return errors.New("retention failureResultTTL must be positive")
	} else if failureResultTTL > maxRetentionFailureResultTTL {
		return fmt.Errorf("retention failureResultTTL must be less than or equal to %s", maxRetentionFailureResultTTL)
	}
	if r.MaxElementsPerRunValue() <= 0 {
		return errors.New("retention maxElementsPerRun must be positive")
	}
	if pageSize := r.ElementPageSizeValue(); pageSize < 1 || pageSize > 100 {
		return errors.New("retention elementPageSize must be in [1, 100]")
	}
	if keepVersionCount := r.KeepVersionCountValue(); keepVersionCount <= 0 {
		return errors.New("retention keepVersionCount must be positive")
	} else if keepVersionCount > maxRetentionKeepVersionCount {
		return fmt.Errorf("retention keepVersionCount must be in [1, %d]", maxRetentionKeepVersionCount)
	}
	if keepVersionDays := r.KeepVersionDaysValue(); keepVersionDays <= 0 {
		return errors.New("retention keepVersionDays must be positive")
	} else if keepVersionDays > maxRetentionKeepVersionDays {
		return fmt.Errorf("retention keepVersionDays must be in [1, %d]", maxRetentionKeepVersionDays)
	}
	if keepOperationDays := r.KeepOperationDaysValue(); keepOperationDays <= 0 {
		return errors.New("retention keepOperationDays must be positive")
	} else if keepOperationDays > maxRetentionKeepOperationDays {
		return fmt.Errorf("retention keepOperationDays must be in [1, %d]", maxRetentionKeepOperationDays)
	}
	return nil
}

type AdminAuth struct {
	SessionSecret     string `toml:"sessionSecret"`
	BootstrapAccount  string `toml:"bootstrapAccount"`
	BootstrapPassword string `toml:"bootstrapPassword"`
	BootstrapNickname string `toml:"bootstrapNickname"`
}

func (a *AdminAuth) HasBootstrapAdmin() bool {
	if a == nil {
		return false
	}
	return a.BootstrapAccount != "" || a.BootstrapPassword != "" || a.BootstrapNickname != ""
}

func (c *CassemAdminConfig) Valid() error {
	if c == nil {
		return errors.New("config is nil")
	}

	if len(c.CassemDBEndpoints) <= 0 {
		return errors.New("empty endpoints")
	}
	if c.Auth == nil || strings.TrimSpace(c.Auth.SessionSecret) == "" {
		return errors.New("empty auth session secret")
	}
	if c.Auth.HasBootstrapAdmin() && (strings.TrimSpace(c.Auth.BootstrapAccount) == "" ||
		strings.TrimSpace(c.Auth.BootstrapPassword) == "" || strings.TrimSpace(c.Auth.BootstrapNickname) == "") {
		return errors.New("incomplete bootstrap admin")
	}
	if c.Retention == nil {
		c.Retention = DefaultRetentionConfig()
	}
	if err := c.Retention.Fix(); err != nil {
		return err
	}

	return nil
}
