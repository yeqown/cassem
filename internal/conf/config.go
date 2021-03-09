package conf

import (
	"io"
	"os"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

type Raft struct {
	RaftBase         string   `toml:"base"`
	RaftBind         string   `toml:"bind"`
	ClusterAddresses []string `toml:"join"` // append to cluster
	ServerId         string   `toml:"serverId"`
}

type HTTP struct {
	Addr  string `toml:"addr"`
	Debug bool
}

type MySQL struct {
	DSN         string `toml:"dsn"`
	MaxIdle     int    `toml:"max_idle"`
	MaxOpen     int    `toml:"max_open"`
	MaxLifeTime int    `toml:"max_life_time"`
	Debug       bool
}

type Config struct {
	Debug bool `toml:"debug"`

	Persistence struct {
		Mysql *MySQL `toml:"mysql"`
	} `toml:"persistence"`

	Server struct {
		HTTP *HTTP `toml:"http"`
		Raft *Raft `toml:"raft"`
	} `toml:"server"`
}

func openFile(path string) (r io.Reader, err error) {
	return os.Open(path)
}

// Load decode into *Config from path (config filepath) in TOML format.
func Load(path string) (*Config, error) {
	if path == "" {
		panic("todo load conf automatically")
	}

	c := new(Config)
	c.Persistence.Mysql = new(MySQL)
	c.Server.HTTP = new(HTTP)
	c.Server.Raft = new(Raft)

	r, err := openFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not open FILE")
	}
	if err = toml.NewDecoder(r).Decode(c); err != nil {
		return nil, errors.Wrap(err, "decode TOML file failed")
	}

	// keep debug mode consistent
	c.Server.HTTP.Debug = c.Debug
	c.Persistence.Mysql.Debug = c.Debug

	return c, nil
}
