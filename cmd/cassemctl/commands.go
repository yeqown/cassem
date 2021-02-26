package main

import (
	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/internal/persistence/mysql"

	"github.com/urfave/cli/v2"
)

func getInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "init",
		Action: func(ctx *cli.Context) error {
			cfg, err := conf.Load(ctx.String("conf"))
			if err != nil {
				return err
			}

			repo, err := mysql.New(cfg.Persistence.Mysql)
			if err != nil {
				return err
			}

			return repo.Migrate()
		},
	}
}
