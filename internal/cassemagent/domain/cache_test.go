package domain

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/yeqown/cassem/internal/concept"
)

type testCacheSuite struct {
	suite.Suite

	cache Cache
}

func (t *testCacheSuite) SetupSuite() {
	t.cache = NewCache(10)
}

func (t testCacheSuite) Test_GetSet() {
	app := "app"
	env := "env"
	key := "key"

	elem, ok := t.cache.Query(app, env, key)
	t.False(ok)
	t.Nil(elem)

	t.cache.Set(app, env, key, &concept.Element{
		Metadata: nil,
		Raw:      []byte("asadasd"),
		Version:  0,
	})

	elem, ok = t.cache.Query(app, env, key)
	t.False(ok)
	t.Nil(elem)

	t.cache.Set(app, env, key, &concept.Element{
		Metadata: nil,
		Raw:      []byte("asadasd"),
		Version:  0,
	})

	elem, ok = t.cache.Query(app, env, key)
	t.True(ok)
	t.Require().NotNil(elem)
	t.NotEmpty(elem.Raw)
}

func (t testCacheSuite) Test_CacheReplacing() {
	// fill cache full
	app := "app"
	env := "env"
	for i := 0; i < 10; i++ {
		t.cache.Set(app, env, "k"+strconv.Itoa(i), &concept.Element{
			Metadata: nil,
			Raw:      []byte(strconv.Itoa(i)),
			Version:  0,
		})
		t.cache.Set(app, env, "k"+strconv.Itoa(i), &concept.Element{
			Metadata: nil,
			Raw:      []byte(strconv.Itoa(i)),
			Version:  0,
		})
	}

	// query check
	for i := 0; i < 10; i++ {
		elem, ok := t.cache.Query(app, env, "k"+strconv.Itoa(i))
		t.True(ok)
		t.Require().NotNil(elem)
		t.Equal(strconv.Itoa(i), string(elem.Raw))
	}

	// "k10" new replaced "k0"
	t.cache.Set(app, env, "k10", &concept.Element{
		Metadata: nil,
		Raw:      []byte("10"),
		Version:  0,
	})
	t.cache.Set(app, env, "k10", &concept.Element{
		Metadata: nil,
		Raw:      []byte("10"),
		Version:  0,
	})

	elem, ok := t.cache.Query(app, env, "k10")
	t.True(ok)
	t.Require().NotNil(elem)
	t.Equal("10", string(elem.Raw))

	elem, ok = t.cache.Query(app, env, "k0")
	t.False(ok)
	t.Require().Nil(elem)
}

func Test_cache(t *testing.T) {
	suite.Run(t, new(testCacheSuite))
}
