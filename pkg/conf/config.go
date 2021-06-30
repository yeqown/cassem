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
	Addr string `toml:"addr"`
}

type BBolt struct {
	Dir string `toml:"dir"`
	DB  string `toml:"db"`
}

type CassemdbConfig struct {
	Persistence struct {
		BBolt *BBolt `toml:"bbolt"`
	} `toml:"persistence"`

	Server struct {
		HTTP *HTTP `toml:"http"`
		Raft *Raft `toml:"raft"`
	} `toml:"server"`
}

func openFile(path string, flag int) (r io.ReadWriteCloser, err error) {
	return os.OpenFile(path, flag, 0644)
}

// Load decode into *CassemdbConfig from path (config filepath) in TOML format.
func Load(path string) (*CassemdbConfig, error) {
	if path == "" {
		panic("todo load conf automatically")
	}

	c := new(CassemdbConfig)
	c.Server.HTTP = new(HTTP)
	c.Server.Raft = new(Raft)

	r, err := openFile(path, os.O_RDONLY)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open `%s`", path)
	}
	defer r.Close()
	if err = toml.NewDecoder(r).Decode(c); err != nil {
		return nil, errors.Wrap(err, "decode TOML file failed")
	}

	return c, nil
}

//
//var defaultConf = &CassemdbConfig{
//	Persistence: struct {
//		BBolt *BBolt `toml:"bbolt"`
//	}{
//		BBolt: &BBolt{
//			Dir: "./bolt",
//			DB:  "cassem.db",
//		},
//	},
//	Server: struct {
//		HTTP *HTTP `toml:"http"`
//		Raft *Raft `toml:"raft"`
//	}{
//		HTTP: &HTTP{
//			Addr: "127.0.0.1:2021",
//		},
//		Raft: &Raft{
//			RaftBase:         "./raft",
//			RaftBind:         "127.0.0.1:3031",
//			ClusterAddresses: []string{},
//			ServerId:         "node1",
//		},
//	},
//}
//
//
//func GenDefaultConfigFile(path string) error {
//	w, err := openFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
//	if err != nil {
//		return errors.Wrapf(err, "could not open `%s`", path)
//	}
//	defer w.Close()
//
//	return toml.NewEncoder(w).Encode(defaultConf)
//}
