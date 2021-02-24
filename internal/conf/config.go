package conf

import (
	"io"
	"os"

	"github.com/yeqown/cassem/internal/persistence/mysql"
	apihtp "github.com/yeqown/cassem/internal/server/api/http"

	"github.com/pelletier/go-toml"
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

func openFile(path string) (r io.Reader, err error) {
	return os.Open(path)
}

// Load decode into *Config from path (config filepath) in TOML format.
func Load(path string) (*Config, error) {
	if path == "" {
		panic("todo load conf automatically")
	}

	c := new(Config)
	c.Persistence.Mysql = new(mysql.ConnectConfig)
	c.Server.HTTP = new(apihtp.Config)

	r, err := openFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not open FILE")
	}
	if err = toml.NewDecoder(r).Decode(c); err != nil {
		return nil, errors.Wrap(err, "decode TOML file failed")
	}

	return c, nil
}
