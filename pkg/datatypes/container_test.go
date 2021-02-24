package datatypes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Container(t *testing.T) {
	c := NewContainer("ns", "container")

	assert.Equal(t, "ns", c.NS())
	assert.Equal(t, "container", c.Key())
	assert.Equal(t, 0, len(c.Fields()))

	_, err := c.SetField(
		NewKVField("string-field", NewPair("ns", "string", WithString("string value"))))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(c.Fields()))
	ok, fld := c.GetField("string-field")
	assert.True(t, ok)
	assert.NotNil(t, fld)
	assert.Equal(t, "string-field", fld.Name())
	assert.Equal(t, KV_FIELD_, fld.Type())

	p, ok := fld.Value().(IPair)
	assert.True(t, ok)
	assert.EqualValues(t, "string value", p.Value())

	_, err = c.SetField(
		NewKVField("", NewPair("ns", "int", WithInt(123))))
	assert.Nil(t, err)

	ok, fld = c.GetField("int")
	assert.True(t, ok)
	assert.NotNil(t, fld)

	p, ok = fld.Value().(IPair)
	assert.True(t, ok)
	assert.EqualValues(t, 123, p.Value())
}

func Test_Container_ToJSON(t *testing.T) {
	c := NewContainer("ns", "container-to-json")

	// construct a complex container and call ToJSON
	expected := `{
	   "b": true,
	   "d": {
	       "df": 1.123,
	       "di": 123,
	       "ds": "string"
	   },
	   "dict": {
	       "b": true,
	       "dict": {
	           "df": 1.123,
	           "di": 123,
	           "ds": "string"
	       },
	       "f": 1.123,
	       "i": 123,
	       "s": "string"
	   },
	   "f": 1.123,
	   "i": 123,
	   "list_basic": [
	       123,
	       1.123,
	       123,
	       true
	   ],
	   "s": "string"
	}`

	s := NewPair("ns", "s", WithString("string"))
	f := NewPair("ns", "f", WithFloat(1.123))
	i := NewPair("ns", "i", WithInt(123))
	b := NewPair("ns", "b", WithBool(true))

	d := WithDict()
	d.Add("ds", s.Value())
	d.Add("df", f.Value())
	d.Add("di", i.Value())
	dictPair := NewPair("ns", "dict", d)

	_, _ = c.SetField(NewKVField("s", s))
	_, _ = c.SetField(NewKVField("f", f))
	_, _ = c.SetField(NewKVField("i", i))
	_, _ = c.SetField(NewKVField("b", b))
	_, _ = c.SetField(NewKVField("d", dictPair))

	listBasic := NewListField("list_basic", []IPair{i, f, i, b})
	_, _ = c.SetField(listBasic)

	dict := NewDictField("dict", map[string]IPair{
		s.Key():        s,
		f.Key():        f,
		i.Key():        i,
		b.Key():        b,
		dictPair.Key(): dictPair,
	})
	_, _ = c.SetField(dict)

	byt, err := c.MarshalJSON()
	assert.Nil(t, err)
	//t.Logf("%s", byt)

	// remove \n\t and space from expected
	expected = strings.Replace(expected, "\n", "", -1)
	expected = strings.Replace(expected, "\t", "", -1)
	expected = strings.Replace(expected, " ", "", -1)
	assert.Equal(t, expected, string(byt))
}
