package conf

import (
	"io"
	"os"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

type Raft struct {
	Base             string   `toml:"base"`
	Bind             string   `toml:"bind"`
	ClusterAddresses []string `toml:"clusterAddresses"`
	BootstrapCluster bool     `toml:"bootstrapCluster"`
}

type HTTP struct {
	Addr string `toml:"addr"`
}

type Bolt struct {
	Dir string `toml:"dir"`
	DB  string `toml:"db"`
}

func openFile(path string, flag int) (r io.ReadWriteCloser, err error) {
	return os.OpenFile(path, flag, 0644)
}

// Load decode into config(struct pointer) from path (config filepath)
// in TOML format.
func Load(path string, c interface{}) (err error) {
	if path == "" {
		panic("todo load conf automatically")
	}

	r, err := openFile(path, os.O_RDONLY)
	if err != nil {
		return errors.Wrapf(err, "could not open `%s`", path)
	}
	defer r.Close()
	if err = toml.NewDecoder(r).Decode(c); err != nil {
		return errors.Wrap(err, "decode TOML file failed")
	}

	return nil
}
