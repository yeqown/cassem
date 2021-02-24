package datatypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewPair(t *testing.T) {
	p := NewPair("ns-pair", "pair-int", WithInt(123123))

	assert.Equal(t, INT_DATATYPE_, p.Value().Datatype())
	assert.Equal(t, IntDT(123123), p.Value().Data())
	assert.Equal(t, "pair-int", p.Key())
	assert.Equal(t, "ns-pair", p.NS())
}

func Test_Pair_MarshalTOML(t *testing.T) {
	p := NewPair("ns", "int-pair", WithInt(123))
	byts, err := p.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, byts)
	t.Logf("int: %s", byts)

	p = NewPair("ns", "float64-pair", WithFloat(123.2132))
	byts, err = p.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, byts)
	t.Logf("float64: %s", byts)

	p = NewPair("ns", "bool-pair", WithBool(true))
	byts, err = p.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, byts)
	t.Logf("bool: %s", byts)

	p = NewPair("ns", "string-pair", WithString("123"))
	byts, err = p.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, byts)
	t.Logf("string: %s", byts)

	// list
	l := WithList()
	l.Append(WithInt(1), WithFloat(1.12312), WithBool(true))
	l2 := WithList()
	l2.Append(WithInt(2), WithString("123123"), l)
	p = NewPair("ns", "list-pair", l2)
	byts, err = p.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, byts)
	t.Logf("list: \n%s", byts)

	d := WithDict()
	d.Add("di", WithInt(1231))
	d.Add("df", WithFloat(1231.1232))
	d.Add("db", WithBool(true))
	d.Add("ds", WithString("12312"))
	d.Add("d2", l)
	d.Add("d3", l2)
	p = NewPair("ns", "dict-pair", d)
	byts, err = p.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, byts)
	t.Logf("dict: \n%s", byts)
}
