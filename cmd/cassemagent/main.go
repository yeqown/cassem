package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/internal/app/agent"
	"github.com/yeqown/cassem/pkg/conf"
)

func isDebug() bool {
	v := os.Getenv("DEBUG")
	return v == "1" || v == "TRUE" || v == "true"
}

func init() {
	log.SetLogLevel(log.LevelInfo)

	if isDebug() {
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
	app.Version = Version
	app.Description = `The storage component of cassem.`
	app.Flags = _cliGlobalFlags
	app.Action = start

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func start(ctx *cli.Context) error {
	c := new(conf.CassemAgentConfig)
	confpath := ctx.String("conf")
	if err := conf.Load(confpath, c); err != nil {
		return err
	}

	log.
		WithFields(log.Fields{
			"conf": c,
			"path": confpath,
		}).
		Debugf("loaded from CONF file: %+v", c)

	d, err := agent.New(c)
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
