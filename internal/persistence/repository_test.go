package persistence_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/mysql"
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/stretchr/testify/suite"
)

type testRepositorySuite struct {
	suite.Suite

	repo persistence.Repository
}

func (s testRepositorySuite) TearDownSuite() {
	// clear testdata
}

func (s testRepositorySuite) Test_Converter() {
	cv := s.repo.Converter()
	s.NotNil(cv)
}

func (s testRepositorySuite) Test_Pair() {
	// from pair
	p := datatypes.NewPair("ns", "kv1", datatypes.WithBool(true))
	v, err := s.repo.Converter().FromPair(p)
	s.Require().Nil(err)
	s.Require().NotNil(v)
	err = s.repo.SavePair(v, false)
	s.Require().Nil(err)

	// read
	v, err = s.repo.GetPair("ns", "kv1")
	s.Require().Nil(err)
	s.Require().NotNil(v)
	// to pair
	pout, err := s.repo.Converter().ToPair(v)
	s.Require().Nil(err)
	s.Require().NotNil(v)
	s.Equal(p, pout)

	// test update
	p2 := datatypes.NewPair("ns", "kv1", datatypes.WithInt(32222))
	v2, _ := s.repo.Converter().FromPair(p2)
	err = s.repo.SavePair(v2, true)
	s.Require().Nil(err)
	// get and check
	v2, err = s.repo.GetPair("ns", "kv1")
	s.Require().Nil(err)
	s.Require().NotNil(v2)
	pout2, err := s.repo.Converter().ToPair(v2)
	s.Require().Nil(err)
	s.Require().NotNil(v)
	s.Equal(p2, pout2)
}

func (s testRepositorySuite) Test_Pair_Paging() {
	out, count, err := s.repo.PagingPairs(nil)
	s.Require().Nil(err)
	s.NotEmpty(count)
	s.NotNil(out)
}

func (s testRepositorySuite) Test_Namespace() {
	err := s.repo.SaveNamespace("")
	s.NotNil(err)
	err = s.repo.SaveNamespace("ns-1")
	s.Nil(err)
	err = s.repo.SaveNamespace("ns-1")
	s.NotNil(err)
	err = s.repo.SaveNamespace("ns-2")
	s.Nil(err)

	out, _, err := s.repo.PagingNamespace(nil)
	s.Nil(err)
	s.GreaterOrEqual(len(out), 2)
}

func (s testRepositorySuite) Test_Container() {
	s2 := datatypes.NewPair("ns", "s", datatypes.WithString("string"))
	f := datatypes.NewPair("ns", "f", datatypes.WithFloat(1.123))
	i := datatypes.NewPair("ns", "i", datatypes.WithInt(123))
	b := datatypes.NewPair("ns", "b", datatypes.WithBool(true))

	d := datatypes.WithDict()
	d.Add("ds", s2.Value())
	d.Add("df", f.Value())
	d.Add("di", i.Value())
	dictPair := datatypes.NewPair("ns", "dict", d)

	// save pairs
	p, err := s.repo.Converter().FromPair(s2)
	s.Require().Nil(err)
	err = s.repo.SavePair(p, true)
	s.Require().Nil(err)
	p, err = s.repo.Converter().FromPair(f)
	s.Require().Nil(err)
	err = s.repo.SavePair(p, true)
	s.Require().Nil(err)
	p, err = s.repo.Converter().FromPair(i)
	s.Require().Nil(err)
	err = s.repo.SavePair(p, true)
	s.Require().Nil(err)
	p, err = s.repo.Converter().FromPair(b)
	s.Require().Nil(err)
	err = s.repo.SavePair(p, true)
	s.Require().Nil(err)
	p, err = s.repo.Converter().FromPair(dictPair)
	s.Require().Nil(err)
	err = s.repo.SavePair(p, true)
	s.Require().Nil(err)

	c := datatypes.NewContainer("ns", "container-1")

	_, _ = c.SetField(datatypes.NewKVField("s", s2))
	_, _ = c.SetField(datatypes.NewKVField("f", f))
	_, _ = c.SetField(datatypes.NewKVField("i", i))
	_, _ = c.SetField(datatypes.NewKVField("b", b))
	_, _ = c.SetField(datatypes.NewKVField("d", dictPair))

	listBasic := datatypes.NewListField("list_basic", []datatypes.IPair{i, i, i, i})
	_, _ = c.SetField(listBasic)

	dict := datatypes.NewDictField("dict", map[string]datatypes.IPair{
		s2.Key():       s2,
		f.Key():        f,
		i.Key():        i,
		b.Key():        b,
		dictPair.Key(): dictPair,
	})
	_, _ = c.SetField(dict)

	// save and read again
	v, err := s.repo.Converter().FromContainer(c)
	s.Require().Nil(err)
	err = s.repo.SaveContainer(v, true)
	s.Require().Nil(err)
	v2, err := s.repo.GetContainer("ns", "container-1")
	s.Require().Nil(err)
	outc, err := s.repo.Converter().ToContainer(v2)
	s.Require().Nil(err)
	//s.EqualValues(c.Fields(), outc.Fields())
	s.True(s.compareContainer(c, outc))

	// remove field, update get and judge
	_, err = c.RemoveField("s")
	s.Require().Nil(err)
	v3, err := s.repo.Converter().FromContainer(c)
	s.Require().Nil(err)
	err = s.repo.SaveContainer(v3, true)
	s.Require().Nil(err)
	v4, err := s.repo.GetContainer("ns", "container-1")
	s.Require().Nil(err)
	outc2, err := s.repo.Converter().ToContainer(v4)
	s.Require().Nil(err)
	//s.EqualValues(c.Fields(), outc2.Fields())
	s.True(s.compareContainer(c, outc2))
}

func (s testRepositorySuite) compareContainer(c1, c2 datatypes.IContainer) (bool, error) {
	byts, err := json.Marshal(c1)
	if err != nil {
		return false, err
	}

	byts2, err := json.Marshal(c2)
	if err != nil {
		return false, err
	}

	ok := bytes.Equal(byts, byts2)
	if !ok {
		s.T().Logf("first:	%s", byts)
		s.T().Logf("second:	%s", byts2)
	}

	return ok, nil
}

func Test_Repo_mysql(t *testing.T) {
	cfg := mysql.ConnectConfig{
		DSN:         "root:@tcp(127.0.0.1:3306)/cassem?charset=utf8mb4&parseTime=true&loc=Local",
		MaxIdle:     10,
		MaxOpen:     100,
		Debug:       true,
		MaxLifeTime: 3600,
	}

	repo, err := mysql.New(&cfg)
	if err != nil {
		t.Fatalf("Test_Repo_mysql failed to open DB")
	}

	//if err = repo.(*mysqlRepo).db.AutoMigrate(
	//	mysql.PairDO{},
	//	mysql.NamespaceDO{},
	//	mysql.ContainerDO{},
	//	mysql.FieldDO{},
	//); err != nil {
	//	t.Fatalf("Test_Repo_mysql failed to AutoMigrate mysql DB: %v", err)
	//}

	s := testRepositorySuite{
		repo: repo,
	}

	suite.Run(t, &s)
}
