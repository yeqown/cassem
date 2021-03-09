package main

import (
	"github.com/yeqown/cassem/internal/authorizer"
	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/internal/persistence/mysql"

	"github.com/urfave/cli/v2"
	"github.com/yeqown/log"
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
			if err = repo.Migrate(); err != nil {
				log.Warn("failed to migrate repo persistence, try again")
			}

			auth, err := authorizer.New(cfg.Persistence.Mysql)
			if err != nil {
				return err
			}
			if err = auth.Migrate(); err != nil {
				log.Warn("failed to migrate auth, try again")
			}

			return nil
		},
	}
}

func getGenConfCommand() *cli.Command {
	return &cli.Command{
		Name:  "genconf",
		Usage: "genconf",
		Action: func(ctx *cli.Context) error {
			// save default conf into current directory.
			return conf.GenDefaultConfigFile(ctx.String("conf"))
		},
	}
}

func getResourceCtlCommands() *cli.Command {
	return &cli.Command{
		Subcommands: func() cli.Commands {
			return cli.Commands{
				addUserCommand(),
			}
		}(),
	}
}

func addUserCommand() *cli.Command {
	return &cli.Command{}
}
