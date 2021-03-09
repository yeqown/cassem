package main

import (
	"os"

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
