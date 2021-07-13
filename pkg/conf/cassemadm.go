package conf

import "errors"

// CassemAdminConfig contains all config to cassemadm.
type CassemAdminConfig struct {
	// CassemDBEndpoints in format like: 127.0.0.1:2021 172.16.2.26:2021 172.16.2.27:2021
	CassemDBEndpoints []string `toml:"cassemdb"`

	// HTTP indicates which port is cassemadm serving on.
	HTTP *HTTP `toml:"http"`
}

func (c *CassemAdminConfig) Valid() error {
	if c == nil {
		return errors.New("config is nil")
	}

	if len(c.CassemDBEndpoints) <= 0 {
		return errors.New("empty endpoints")
	}

	return nil
}
