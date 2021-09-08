package conf

import "github.com/pkg/errors"

// CassemAgentConfig contains all config to cassemadm.
type CassemAgentConfig struct {
	// CassemDBEndpoints in format like: 127.0.0.1:2021 172.16.2.26:2021 172.16.2.27:2021
	CassemDBEndpoints []string `toml:"cassemdb"`

	// Server indicates which port is cassemadm serving on.
	Server *Server `toml:"server"`

	// TTL indicates how much time to live for agents registrations.
	TTL int32 `toml:"ttl"`

	// RenewInterval indicates how much time to renew agents registrations.
	// Make sure that RenewInterval is less than TTL.
	// actual renew interval will be calculated while agent start as the following expression:
	// actualRenewInterval = RenewInterval + randn(TTL - RenewInterval)
	RenewInterval int32 `toml:"renewInterval"`

	// ElementCacheSize represents how many item can be cached in this agent node. notice
	// that 'app-env-elemKey' represents a unique item.
	ElementCacheSize int32 `toml:"elementCacheSize"`
}

func (c *CassemAgentConfig) Valid() error {
	if c == nil {
		return errors.New("config is nil")
	}

	if c.TTL == 0 {
		c.TTL = 30
	}
	if c.RenewInterval == 0 {
		c.TTL = int32(float32(c.TTL) * 0.66666667)
	}

	if c.ElementCacheSize == 0 {
		c.ElementCacheSize = 1000
	}

	if c.RenewInterval > c.TTL {
		return errors.New("renewInterval should be lte than TTL")
	}

	if len(c.CassemDBEndpoints) <= 0 {
		return errors.New("empty endpoints")
	}

	return nil
}
