package app_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/yeqown/cassem/internal/cassemadm/app"
	"github.com/yeqown/cassem/pkg/conf"
)

type appTestSuite struct {
	suite.Suite

	coord app.ICoordinator
	ctx   context.Context
}

func (a *appTestSuite) SetupSuite() {
	var err error
	a.coord, err = app.New(&conf.CassemAdminConfig{
		CassemDBCluster: "cassemdb://auth/127.0.0.1:2021,127.0.0.1:2022,127.0.0.1:2023",
		// HTTP:            nil,
	})
	a.ctx = context.TODO()

	if err != nil {
		panic(err)
	}
}

func (a appTestSuite) Test_CreateElement() {
	err := a.coord.CreateElement(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
		[]byte("this is a text"),
	)
	a.NoError(err)
}

func (a appTestSuite) Test_GetElement() {
	elt, err := a.coord.GetElement(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
		0,
	)
	a.NoError(err)
	a.T().Logf("%+v", elt.Version)
	a.T().Logf("%+v", elt.Metadata)
	a.T().Logf("%v", elt.Raw)
}

func (a appTestSuite) Test_UpdateElement() {
	err := a.coord.UpdateElement(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
		[]byte("this is a text222"),
	)
	a.NoError(err)
}

func (a appTestSuite) Test_DeleteElement() {
	err := a.coord.DeleteElement(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
	)
	a.NoError(err)
}

func Test_App(t *testing.T) {
	suite.Run(t, new(appTestSuite))
}
