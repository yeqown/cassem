package main

import (
	"os"

	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/internal/server/daemon"

	"github.com/urfave/cli/v2"
	"github.com/yeqown/log"
)

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "cassemd"
	app.Usage = "cassem daemon server"
	app.Authors = []*cli.Author{
		{
			Name:  "yeqown",
			Email: "yeqown@gmail.com",
		},
	}
	app.Version = "v1.6.4"
	app.Description = `The server of cassem.`
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

	// DONE(@yeqown) use CLI args override cfg.Server.Raft
	cfg.Server.Raft.Bind = ctx.String("bind")
	cfg.Server.Raft.Base = ctx.String("raft-base")
	cfg.Server.Raft.Join = ctx.String("join")
	cfg.Server.Raft.Listen = ctx.String("raft-listen")
	cfg.Server.Raft.ServerID = ctx.String("id")
	cfg.Server.HTTP.Addr = ctx.String("http-listen")

	log.Debugf("config %+v", cfg.Server.Raft)

	d, err := daemon.New(cfg)
	if err != nil {
		return err
	}

	d.Heartbeat()

	return nil
}

var _cliGlobalFlags = []cli.Flag{
	&cli.StringFlag{
		Name:        "conf",
		Aliases:     []string{"c"},
		Value:       "./configs/cassem.example.toml",
		DefaultText: "./configs/cassem.example.toml",
		Usage:       "choose which `path/to/file` to load",
		Required:    false,
	},
	&cli.StringFlag{
		Name:        "raft-base",
		Value:       "./debugdata/d1",
		DefaultText: "./debugdata/d1",
		Usage:       "raft consensus protocol component's base directory",
		Required:    false,
	},
	&cli.StringFlag{
		Name:        "raft-listen",
		Value:       "127.0.0.1:4021",
		DefaultText: "127.0.0.1:4021",
		Usage:       "raft consensus protocol cluster address",
		Required:    false,
	},
	&cli.StringFlag{
		Name:        "http-listen",
		Value:       "127.0.0.1:2021",
		DefaultText: "127.0.0.1:2021",
		Usage:       "http server listen on",
		Required:    false,
	},
	&cli.StringFlag{
		Name:        "bind",
		Value:       "127.0.0.1:3021",
		DefaultText: "127.0.0.1:3021",
		Usage:       "raft consensus protocol component's used address",
		Required:    false,
	},
	&cli.StringFlag{
		Name:        "join",
		Value:       "",
		DefaultText: "",
		Usage:       "raft consensus protocol cluster address",
		Required:    false,
	},
	&cli.StringFlag{
		Name:        "id",
		Value:       "",
		DefaultText: "",
		Usage:       "server identify name, should be unique",
		Required:    true,
	},
}
