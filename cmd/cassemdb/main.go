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
}
