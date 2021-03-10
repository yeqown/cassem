package main

import (
	"os"
	"sync"

	"github.com/yeqown/cassem/internal/conf"

	"github.com/urfave/cli/v2"
	"github.com/yeqown/log"
)

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "cassem"
	app.Usage = "cassem control tool"
	app.Authors = []*cli.Author{
		{
			Name:  "yeqown",
			Email: "yeqown@gmail.com",
		},
	}
	app.Version = "v1.0.0"
	app.Description = `A tool for managing cassem`
	app.Flags = _cliGlobalFlags

	mountCommands(app)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func mountCommands(app *cli.App) {
	app.Commands = []*cli.Command{
		getInitCommand(),
		getGenConfCommand(),
		getResourceCtlCommands(),
		//getClusterDashCommand(),
	}
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
}

var (
	// _config DO NOT use this directly !!! should only access it by getConfig.
	_config *conf.Config
	_err    error

	_once sync.Once
)

// getConfig will be called multiple times but only execute once.
func getConfig(ctx *cli.Context) (*conf.Config, error) {
	_once.Do(func() {
		_config, _err = conf.Load(ctx.String("conf"))
	})

	return _config, _err
}
