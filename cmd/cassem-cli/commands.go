package main

import (
	"github.com/yeqown/cassem/internal/conf"
	coord "github.com/yeqown/cassem/internal/coordinator"
	mysqlimpl "github.com/yeqown/cassem/internal/persistence/mysql"
	apihttp "github.com/yeqown/cassem/internal/server/api/http"

	"github.com/urfave/cli/v2"
)

// TODO(@yeqown) fill this command
func getInitCommand() *cli.Command {
	return &cli.Command{}
}

func getServerCommand() *cli.Command {
	return &cli.Command{
		Name:     "serve",
		Usage:    "start the server",
		Category: "server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "addr",
				Value:       ":2021",
				DefaultText: ":2021",
				Required:    false,
				Usage:       "`addr` is the host:port of server listening on",
			},
		},
		ArgsUsage: "--addr `HOST:PORT`",
		Action: func(c *cli.Context) error {
			// load config
			cpath := c.String("conf")
			cfg, err := conf.Load(cpath)
			if err != nil {
				return err
			}

			repo, err := mysqlimpl.New(cfg.Persistence.Mysql)
			if err != nil {
				return err
			}

			// start server
			addr := c.String("addr")
			return apihttp.New(addr, coord.New(repo)).ListenAndServe()
		},
	}
}
