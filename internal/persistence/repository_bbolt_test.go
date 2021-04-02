package persistence_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/yeqown/cassem/internal/persistence/bbolt"

	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/stretchr/testify/suite"
)

type testRepositoryBBoltSuite struct {
	suite.Suite

	repo      persistence.Repository
	convertor persistence.Converter
}

func (s testRepositoryBBoltSuite) TearDownSuite() {
	// clear testdata
}

func (s testRepositoryBBoltSuite) Test_Converter() {
	cv := bbolt.NewConverter()
	s.NotNil(cv)
}

func (s testRepositoryBBoltSuite) Test_Pair() {
	// from pair
	p := datatypes.NewPair("ns", "kv1", datatypes.WithBool(true))
	v, err := s.convertor.FromPair(p)
	s.Require().Nil(err)
	s.Require().NotNil(v)
	err = s.repo.SavePair(v, false)
	s.Require().Nil(err)

	// read
	v, err = s.repo.GetPair("ns", "kv1")
	s.Require().Nil(err)
	s.Require().NotNil(v)
	// to pair
	pout, err := s.convertor.ToPair(v)
	s.Require().Nil(err)
	s.Require().NotNil(v)
	s.Equal(p, pout)

	// test update
	p2 := datatypes.NewPair("ns", "kv1", datatypes.WithInt(32222))
	v2, _ := s.convertor.FromPair(p2)
	err = s.repo.SavePair(v2, true)
	s.Require().Nil(err)
	// get and check
	v2, err = s.repo.GetPair("ns", "kv1")
	s.Require().Nil(err)
	s.Require().NotNil(v2)
	pout2, err := s.convertor.ToPair(v2)
	s.Require().Nil(err)
	s.Require().NotNil(v)
	s.Equal(p2, pout2)
}

func (s testRepositoryBBoltSuite) Test_Pair_Paging() {
	out, count, err := s.repo.PagingPairs(&persistence.PagingPairsFilter{
		Limit:      10,
		Offset:     0,
		KeyPattern: "",
		Namespace:  "ns",
	})
	s.Require().Nil(err)
	s.NotEmpty(count)
	s.NotNil(out)
}

func (s testRepositoryBBoltSuite) Test_Namespace() {
	err := s.repo.SaveNamespace("")
	s.NotNil(err)
	err = s.repo.SaveNamespace("ns-1")
	s.Nil(err)
	err = s.repo.SaveNamespace("ns-1")
	s.Nil(err)
	err = s.repo.SaveNamespace("ns")
	s.Nil(err)

	out, _, err := s.repo.PagingNamespace(&persistence.PagingNamespacesFilter{
		Limit:            10,
		Offset:           0,
		NamespacePattern: "",
	})
	s.Nil(err)
	s.GreaterOrEqual(len(out), 2)

	out2, _, err2 := s.repo.PagingNamespace(&persistence.PagingNamespacesFilter{
		Limit:            2,
		Offset:           1,
		NamespacePattern: "",
	})
	s.Nil(err2)
	s.GreaterOrEqual(len(out2), 1)
}

func (s testRepositoryBBoltSuite) Test_Container() {
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
	p, err := s.convertor.FromPair(s2)
	s.Require().Nil(err)
	err = s.repo.SavePair(p, true)
	s.Require().Nil(err)
	p, err = s.convertor.FromPair(f)
	s.Require().Nil(err)
	err = s.repo.SavePair(p, true)
	s.Require().Nil(err)
	p, err = s.convertor.FromPair(i)
	s.Require().Nil(err)
	err = s.repo.SavePair(p, true)
	s.Require().Nil(err)
	p, err = s.convertor.FromPair(b)
	s.Require().Nil(err)
	err = s.repo.SavePair(p, true)
	s.Require().Nil(err)
	p, err = s.convertor.FromPair(dictPair)
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
	v, err := s.convertor.FromContainer(c)
	s.Require().Nil(err)
	err = s.repo.SaveContainer(v, true)
	s.Require().Nil(err)
	v2, err := s.repo.GetContainer("ns", "container-1")
	s.Require().Nil(err)
	outc, err := s.convertor.ToContainer(v2)
	s.Require().Nil(err)
	//s.EqualValues(c.Fields(), outc.Fields())
	s.True(s.compareContainer(c, outc))

	// remove field, update get and judge
	_, err = c.RemoveField("s")
	s.Require().Nil(err)
	v3, err := s.convertor.FromContainer(c)
	s.Require().Nil(err)
	err = s.repo.SaveContainer(v3, true)
	s.Require().Nil(err)
	v4, err := s.repo.GetContainer("ns", "container-1")
	s.Require().Nil(err)
	outc2, err := s.convertor.ToContainer(v4)
	s.Require().Nil(err)
	//s.EqualValues(c.Fields(), outc2.Fields())
	s.True(s.compareContainer(c, outc2))
}

func (s testRepositoryBBoltSuite) compareContainer(c1, c2 datatypes.IContainer) (bool, error) {
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

func (s testRepositoryBBoltSuite) Test_RepositoryUser() {
	err := s.repo.CreateUser(&persistence.User{
		Account: "root",
		// 123456
		PasswordWithSalt: "92f9ce613443bfa68e8d511ed579d0e29fe69778de19ab4dda10a35360940882",
		Name:             "cassem",
	})
	s.Nil(err)
}

func Test_Repo_BBolt_mysql(t *testing.T) {
	cfg := conf.BBolt{
		Dir: "./debugdata",
		DB:  "cassem.db",
	}

	repo, err := bbolt.New(&cfg)
	if err != nil {
		t.Fatalf("Test_Repo_BBolt_mysql failed to open DB: %v", err)
	}

	if err := repo.Migrate(); err != nil {
		t.Fatalf("Test_Repo_BBolt_mysql failed to migrate DB: %v", err)
	}

	s := testRepositoryBBoltSuite{
		repo:      repo,
		convertor: bbolt.NewConverter(),
	}

	suite.Run(t, &s)
}
