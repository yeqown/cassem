package main

import (
	"fmt"
	"os"
	"time"

	cassemdb "github.com/yeqown/cassem/internal/cassemdb/app"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/runtime"

	"github.com/urfave/cli/v2"
	"github.com/yeqown/log"
)

func init() {
	log.SetLogLevel(log.LevelInfo)
	log.SetTimeFormat(true, time.RFC3339)

	if runtime.IsDebug() {
		log.SetCallerReporter(true)
		log.SetLogLevel(log.LevelDebug)
	}
}

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "cassemdb"
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
	c := new(conf.CassemdbConfig)
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

	d, err := cassemdb.New(c)
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
		Value:       "./configs/cassemdb.example.toml",
		DefaultText: "./configs/cassemdb.example.toml",
		Usage:       "choose which `path/to/file` to load",
		Required:    false,
	},
	&cli.StringFlag{
		Name:        "storage",
		Value:       "./storage",
		DefaultText: "./storage",
		Usage:       "specify the directory to store cassemdb data",
		Required:    false,
	},
	&cli.StringFlag{
		Name:        "endpoint",
		Value:       "0.0.0.0:2021",
		DefaultText: "0.0.0.0:2021",
		Usage:       "specify the endpoint to connect to",
		Required:    false,
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
func fixConfig(ctx *cli.Context, c *conf.CassemdbConfig) {
	base := ctx.String("storage")
	if base == "" && c.Raft.Base == "" {
		base = "./storage"
	}
	if base != "" {
		c.Bolt.Dir = base
		c.Raft.Base = base
	}

	if endpoint := ctx.String("endpoint"); endpoint != "" {
		c.Addr = endpoint
	}

	if bind := ctx.String("raft.bind"); bind != "" {
		c.Raft.Bind = bind
	}

	if cluster := ctx.String("raft.cluster"); cluster != "" {
		c.Raft.Cluster = cluster
	}
}
