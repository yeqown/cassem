package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"github.com/yeqown/log"

	cassemagent "github.com/yeqown/cassem/internal/cassemagent/app"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/runtime"
)

func init() {
	log.SetLogLevel(log.LevelInfo)

	if runtime.IsDebug() {
		log.SetCallerReporter(true)
		log.SetLogLevel(log.LevelDebug)
	}
}

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "cassemagent"
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
	c := new(conf.CassemAgentConfig)
	if err := conf.Load(ctx.String("conf"), c); err != nil {
		return err
	}

	log.
		WithFields(log.Fields{
			"conf": c,
		}).
		Debugf("loaded from CONF file: %+v", c)

	d, err := cassemagent.New(c)
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
		Value:       "./configs/cassemagent.example.toml",
		DefaultText: "./configs/cassemagent.example.toml",
		Usage:       "choose which `path/to/file` to load",
		Required:    false,
	},
}
