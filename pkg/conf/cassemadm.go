package conf

import (
	"errors"
	"strings"
)

// CassemAdminConfig contains all config to cassemadm.
type CassemAdminConfig struct {
	// CassemDBEndpoints in format like: 127.0.0.1:2021 172.16.2.26:2021 172.16.2.27:2021
	CassemDBEndpoints []string `toml:"cassemdb"`

	// HTTP indicates which port is cassemadm serving on.
	HTTP *Server `toml:"http"`

	Auth *AdminAuth `toml:"auth"`
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

	return nil
}
