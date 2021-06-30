package main

import (
	"os"

	"github.com/yeqown/cassem/apps/cassemdb"
	"github.com/yeqown/cassem/internal/conf"

	"github.com/urfave/cli/v2"
	"github.com/yeqown/log"
)

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
	app.Version = "v1.0.0"
	app.Description = `The storage component of cassem.`
	app.Flags = _cliGlobalFlags
	app.Action = start

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func start(ctx *cli.Context) error {
	cfg, err := conf.Load(ctx.String("conf"))
	if err != nil {
		return err
	}

	log.Debugf("loaded from CONF file: %+v", cfg)
	log.Debugf("server.raft: %+v", cfg.Server.Raft)

	d, err := cassemdb.New(cfg)
	if err != nil {
		return err
	}

	// run as daemon
	d.Heartbeat()

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
