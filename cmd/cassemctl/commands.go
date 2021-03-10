package main

import (
	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/mysql"

	"github.com/urfave/cli/v2"
	"github.com/yeqown/log"
)

func getRepository(ctx *cli.Context) (persistence.Repository, error) {
	c, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}

	repo, err := mysql.New(c.Persistence.Mysql)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func getInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "init",
		Action: func(ctx *cli.Context) error {
			repo, err := getRepository(ctx)
			if err != nil {
				return err
			}
			if err = repo.Migrate(); err != nil {
				log.Warn("failed to migrate repo persistence, try again")
			}

			auth, err := getAuthorizer(ctx)
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
		Name:  "resource",
		Usage: "resource",
		Subcommands: func() cli.Commands {
			return cli.Commands{
				addUserCommand(),
				resetUserPasswordCommand(),
				//listUserPolicyCommand(),
			}
		}(),
	}
}
