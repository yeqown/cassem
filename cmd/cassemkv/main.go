package main

import (
	"fmt"
	"os"
	"time"

	"github.com/yeqown/cassem/internal/app/kv"
	"github.com/yeqown/cassem/pkg/conf"

	"github.com/urfave/cli/v2"
	"github.com/yeqown/log"
)

func isDebug() bool {
	v := os.Getenv("DEBUG")
	return v == "1" || v == "TRUE" || v == "true"
}

func init() {
	log.SetLogLevel(log.LevelInfo)
	log.SetTimeFormat(true, time.RFC3339)

	if isDebug() {
		log.SetCallerReporter(true)
		log.SetLogLevel(log.LevelDebug)
	}
}

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "cassemkv"
	app.Usage = "cassem storage server"
	app.Authors = []*cli.Author{
		{
			Name:  "yeqown",
			Email: "yeqown@gmail.com",
		},
	}
	app.Version = fmt.Sprintf(`{"version": %s, "buildTime": %s, "gitHash": %s}`, Version, BuildTime, GitHash)
	app.Description = `The storage component of cassem.`
	app.Flags = _cliGlobalFlags
	app.Action = start

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func start(ctx *cli.Context) error {
	c := new(conf.CassemKVConfig)
	if err := conf.Load(ctx.String("conf"), c); err != nil {
		return err
	}

	fixConfig(ctx, c)

	log.
		WithFields(log.Fields{
			"conf":      c,
			"conf.raft": c.Raft,
			"conf.bolt": c.Bolt,
		}).
		Debugf("loaded from CONF file: %+v", c)

	d, err := kv.New(c)
	if err != nil {
		return err
	}

	d.Run()

	return nil
}

var _cliGlobalFlags = []cli.Flag{
	&cli.StringFlag{
		Name:        "conf",
		Aliases:     []string{"c"},
		Value:       "./configs/cassemkv.example.toml",
		DefaultText: "./configs/cassemkv.example.toml",
		Usage:       "choose which `path/to/file` to load",
		Required:    false,
	},
	&cli.StringFlag{
		Name:        "storage",
		Value:       "./storage",
		DefaultText: "./storage",
		Usage:       "specify the directory to store cassemkv data",
		Required:    false,
	},
	&cli.StringFlag{
		Name:     "listen-addr",
		Usage:    "specify the cassemkv gRPC listen address",
		Required: true,
	},
	&cli.StringFlag{
		Name:     "advertise-addr",
		Usage:    "specify the cassemkv gRPC endpoint advertised to clients",
		Required: true,
	},
	&cli.StringFlag{
		Name:     "raft.cluster",
		Value:    "",
		Usage:    "specify all of the cluster nodes urls, split by comma ','",
		Required: false,
	},
	&cli.StringFlag{
		Name:     "raft.bind",
		Value:    "",
		Usage:    "specify the address of current node to serve raft server.",
		Required: false,
	},
}

// fixConfig get nodeId while could not find in config file, next step to determine
// the value from ENV and flags by order.
func fixConfig(ctx *cli.Context, c *conf.CassemKVConfig) {
	base := ctx.String("storage")
	if base == "" && c.Raft.Base == "" {
		base = "./storage"
	}
	if base != "" {
		c.Bolt.Dir = base
		c.Raft.Base = base
	}

	if listenAddr := ctx.String("listen-addr"); listenAddr != "" {
		c.ListenAddr = listenAddr
	}

	if advertiseAddr := ctx.String("advertise-addr"); advertiseAddr != "" {
		c.AdvertiseAddr = advertiseAddr
	}

	if bind := ctx.String("raft.bind"); bind != "" {
		c.Raft.Bind = bind
	}

	if cluster := ctx.String("raft.cluster"); cluster != "" {
		c.Raft.Cluster = cluster
	}
}
