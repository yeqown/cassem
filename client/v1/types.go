package v1

import (
	"time"

	pb "github.com/yeqown/cassem/clientv1/gen"
)

var defaultConf = Config{
	Endpoint:    "",
	DialTimeout: 5 * time.Second,
	Watching:    nil,
}

type Config struct {
	Endpoint    string
	DialTimeout time.Duration
	Watching    []WatchContainerOption
	Fn          HandlerFunc
}

type WatchContainerOption struct {
	Namespace string
	Keys      []string
	Format    ContainerFormat
}

func adaptConfig(c *Config) *Config {
	if c == nil {
		c = &defaultConf
	}

	if c.DialTimeout <= 0 {
		c.DialTimeout = 5 * time.Second
	}

	if c.Fn == nil {
		c.Fn = defaultChangeHandlerFunc
	}

	return c
}

// Changes is a copy of github.com/yeqown/cassem/internal/watcher.Changes.
type Changes struct {
	Key       string
	Namespace string
	Format    ContainerFormat
	CheckSum  string
	Data      []byte
}

type ContainerFormat string

const (
	JSON ContainerFormat = "json"
	TOML ContainerFormat = "toml"
)

// toPBFormat it panics while format could not be handled.
func toPBFormat(format ContainerFormat) pb.Format {
	switch format {
	case JSON:
		return pb.Format_JSON
	case TOML:
		return pb.Format_TOML
	}

	panic("unsupported datatypes format")
}

// fromPBFormat it panics while format could not be handled.
func fromPBFormat(format pb.Format) ContainerFormat {
	switch format {
	case pb.Format_JSON:
		return JSON
	case pb.Format_TOML:
		return TOML
	}

	panic("unsupported pb format")
}

func (c ContainerFormat) String() string {
	return string(c)
}
