package concept

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type appTestSuite struct {
	suite.Suite

	hybrid Hybrid
	ctx    context.Context
}

func (a *appTestSuite) SetupSuite() {
	var err error
	a.hybrid, err = NewHybrid([]string{"127.0.0.1:2021", "127.0.0.1:2022", "127.0.0.1:2023"})
	if err != nil {
		panic(err)
	}
	a.ctx = context.TODO()

}

func (a appTestSuite) Test_CreateElement() {
	err := a.hybrid.CreateElement(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
		[]byte("this is a text"),
		RawContentType_PLAINTEXT,
	)
	a.NoError(err)
}

func (a appTestSuite) Test_UpdateElement() {
	err := a.hybrid.UpdateElement(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
		[]byte("this is a text, updated"),
	)
	a.NoError(err)
}

func (a appTestSuite) Test_GetElementLatest() {
	elt, err := a.hybrid.GetElementWithVersion(
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

func (a appTestSuite) Test_GetElementWithVersion() {
	elt, err := a.hybrid.GetElementWithVersion(
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

func (a appTestSuite) Test_GetElement_NotExists() {
	elt, err := a.hybrid.GetElementWithVersion(
		a.ctx,
		"app",
		"env",
		"Test_CreateElement",
		99,
	)

	a.Error(err)
	a.Nil(elt)
}

func (a appTestSuite) Test_DeleteElement() {
	err := a.hybrid.DeleteElement(
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
