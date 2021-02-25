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
		Required:    true,
	},
}
