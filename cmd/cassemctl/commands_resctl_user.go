package main

import (
	"encoding/json"

	"github.com/yeqown/cassem/internal/authorizer"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var _userMetaDataFlag = &cli.StringFlag{
	Name:     "data",
	Aliases:  []string{"d"},
	Required: true,
}

type userMetadata struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func getUserMetadata(ctx *cli.Context) (*userMetadata, error) {
	data := ctx.String("d")
	md := new(userMetadata)
	if err := json.Unmarshal([]byte(data), md); err != nil {
		return nil, errors.Wrapf(err, "could not parse user data")
	}

	return md, nil
}

func getAuthorizer(ctx *cli.Context) (authorizer.IAuthorizer, error) {
	cfg, err := getConfig(ctx)
	if err != nil {
		return nil, err
	}

	auth, err := authorizer.New(cfg.Persistence.Mysql)
	if err != nil {
		return nil, err
	}

	return auth, nil
}

func addUserCommand() *cli.Command {
	return &cli.Command{
		Name:     "adduser",
		Usage:    "provide the new account data in terminal with JSON format",
		Category: "user",
		Flags:    []cli.Flag{_userMetaDataFlag},
		Action: func(ctx *cli.Context) error {
			auth, err := getAuthorizer(ctx)
			if err != nil {
				return err
			}

			md, err := getUserMetadata(ctx)
			if err != nil {
				return err
			}

			return auth.AddUser(md.Account, md.Password, md.Name)
		},
	}
}

func resetUserPasswordCommand() *cli.Command {
	return &cli.Command{
		Name:     "resetpwd",
		Usage:    "provide the new account data in terminal with JSON format",
		Category: "user",
		Flags:    []cli.Flag{_userMetaDataFlag},
		Action: func(ctx *cli.Context) error {
			auth, err := getAuthorizer(ctx)
			if err != nil {
				return err
			}

			md, err := getUserMetadata(ctx)
			if err != nil {
				return err
			}

			return auth.ResetPassword(md.Account, md.Password)
		},
	}
}

func listUserPolicyCommand() *cli.Command {
	return &cli.Command{
		Name:     "listpolicy",
		Usage:    "provide the new account data in terminal with JSON format",
		Category: "user",
		Flags:    []cli.Flag{_userMetaDataFlag},
		Action: func(ctx *cli.Context) error {
			panic("implement me")
		},
	}
}
