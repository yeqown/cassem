package conf

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"

	"github.com/yeqown/cassem/pkg/runtime"
)

type Raft struct {
	Base string `toml:"-"`
	Bind string `toml:"bind"`
	// Cluster "http://node1:3021,http://node2:3021,http://node3:3021", if the cluster is empty,
	// cluster is "$Bind" as default value.
	Cluster string `toml:"cluster"`
	// NodeID is the index of the Bind address in Cluster.
	NodeID    uint64 `toml:"-"`
	SnapCount uint   `toml:"snapCount"`
}

func (r *Raft) Fix() error {
	if r.Bind == "" {
		return fmt.Errorf("raft.bind address is required")
	}

	if r.Cluster == "" {
		r.Cluster = r.Bind
	}
	arr := strings.Split(r.Cluster, ",")
	pos := runtime.IndexOf(r.Bind, arr)
	if pos < 0 {
		return fmt.Errorf("raft.bind could not found in raft.cluster")
	}

	r.NodeID = uint64(pos + 1)
	return nil
}

type Server struct {
	// Addr of server in format of: HOST:PORT with default scheme TCP.
	Addr string `toml:"addr"`
}

type Bolt struct {
	Dir string `toml:"-"`
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
