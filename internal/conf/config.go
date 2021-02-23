package conf

import (
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/yeqown/cassem/internal/persistence/mysql"
)

type Config struct {
	Persistence struct {
		Mysql *mysql.ConnectConfig `toml:"mysql"`
	} `toml:"persistence"`
}

func Load(p string) (*Config, error) {
	if p == "" {
		panic("todo load conf automatically")
	}

	c := new(Config)

	if _, err := toml.DecodeFile(p, c); err != nil {
		return nil, errors.Wrap(err, "could not decode FILE in TOML format")
	}

	return c, nil
}
