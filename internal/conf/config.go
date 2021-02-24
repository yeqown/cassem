package conf

import (
	"github.com/yeqown/cassem/internal/persistence/mysql"
	apihtp "github.com/yeqown/cassem/internal/server/api/http"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type Config struct {
	Persistence struct {
		Mysql *mysql.ConnectConfig `toml:"mysql"`
	} `toml:"persistence"`

	Server struct {
		HTTP *apihtp.Config `toml:"http"`
	} `toml:"server"`
}

func Load(p string) (*Config, error) {
	if p == "" {
		panic("todo load conf automatically")
	}

	c := new(Config)
	c.Persistence.Mysql = new(mysql.ConnectConfig)
	c.Server.HTTP = new(apihtp.Config)

	if _, err := toml.DecodeFile(p, c); err != nil {
		return nil, errors.Wrap(err, "could not decode FILE in TOML format")
	}

	return c, nil
}
