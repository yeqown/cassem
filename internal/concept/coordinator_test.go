package concept

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type coordinatorTestSuite struct {
	suite.Suite

	agg AdmAggregate
	ctx context.Context
}

func (a *coordinatorTestSuite) SetupSuite() {
	var err error
	a.ctx = context.TODO()
	endpoints := []string{"127.0.0.1:2021", "127.0.0.1:2022", "127.0.0.1:2023"}
	a.agg, err = NewAdmAggregate(endpoints)
	if err != nil {
		panic(err)
	}
}

func (a coordinatorTestSuite) Test_CreateElement() {
	err := a.agg.CreateElement(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
		[]byte("this is a text"),
		ContentType_PLAINTEXT,
	)
	a.NoError(err)
}

func (a coordinatorTestSuite) Test_UpdateElement() {
	err := a.agg.UpdateElement(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
		[]byte("this is a text, updated"),
	)
	a.NoError(err)
}

func (a coordinatorTestSuite) Test_GetElementLatest() {
	elt, err := a.agg.GetElementWithVersion(
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

func (a coordinatorTestSuite) Test_GetElementWithVersion() {
	elt, err := a.agg.GetElementWithVersion(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
		1,
	)
	a.NoError(err)
	a.T().Logf("%+v", elt.Version)
	a.T().Logf("%+v", elt.Metadata)
	a.T().Logf("%v", elt.Raw)
}

func (a coordinatorTestSuite) Test_GetElement_NotExists() {
	elt, err := a.agg.GetElementWithVersion(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
		99,
	)

	a.Error(err)
	a.Nil(elt)
}

func (a coordinatorTestSuite) Test_DeleteElement() {
	err := a.agg.DeleteElement(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
	)
	a.NoError(err)
}

func (a coordinatorTestSuite) Test_RegisterInstance() {
	err := a.agg.RegisterInstance(a.ctx, &Instance{
		ClientId:           "clientId",
		Ip:                 "172.168.1.1",
		App:                "app",
		Env:                "env",
		WatchKeys:          []string{"k1", "k2", "k3"},
		LastRenewTimestamp: 0,
	})
	a.NoError(err)
}

func (a coordinatorTestSuite) Test_RenewInstance() {
	err := a.agg.RenewInstance(a.ctx, &Instance{
		ClientId:           "clientId",
		Ip:                 "172.168.1.1",
		App:                "app",
		Env:                "env",
		WatchKeys:          []string{"k1", "k2", "k3"},
		LastRenewTimestamp: 0,
	})
	a.NoError(err)
}

func (a coordinatorTestSuite) Test_UnregisterInstance() {
	ins := &Instance{
		ClientId:           "clientId",
		Ip:                 "172.168.1.1",
		App:                "app",
		Env:                "env",
		WatchKeys:          []string{"k1", "k2", "k3"},
		LastRenewTimestamp: 0,
	}
	err := a.agg.UnregisterInstance(a.ctx, ins.Id())
	a.NoError(err)
}

func Test_App(t *testing.T) {
	suite.Run(t, new(coordinatorTestSuite))
}
