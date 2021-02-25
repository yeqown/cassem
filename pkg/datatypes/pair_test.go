package datatypes

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func Test_NewPair(t *testing.T) {
	p := NewPair("ns-pair", "pair-int", WithInt(123123))

	assert.Equal(t, INT_DATATYPE_, p.Value().Datatype())
	assert.Equal(t, IntDT(123123), p.Value().Data())
	assert.Equal(t, "pair-int", p.Key())
	assert.Equal(t, "ns-pair", p.NS())
}

func encodeAndTest(t *testing.T, expected string, v interface{}) {
	byts, err := json.Marshal(v)
	require.Nil(t, err)
	require.NotEmpty(t, byts)

	output := string(byts)
	// remove \n\t and space from expected
	expected = strings.Replace(expected, "\n", "", -1)
	expected = strings.Replace(expected, "\t", "", -1)
	expected = strings.Replace(expected, " ", "", -1)

	output = strings.Replace(output, "\n", "", -1)
	output = strings.Replace(output, "\t", "", -1)
	output = strings.Replace(output, " ", "", -1)

	assert.Equal(t, expected, output)
}

func Test_Pair_ToMarshalInterface(t *testing.T) {
	p := NewPair("ns", "int-pair", WithInt(123))
	v := p.ToMarshalInterface()
	encodeAndTest(t, "123", v)

	p = NewPair("ns", "float64-pair", WithFloat(123.2132))
	v = p.ToMarshalInterface()
	encodeAndTest(t, "123.2132", v)

	p = NewPair("ns", "bool-pair", WithBool(true))
	v = p.ToMarshalInterface()
	encodeAndTest(t, "true", v)

	p = NewPair("ns", "string-pair", WithString("123"))
	v = p.ToMarshalInterface()
	encodeAndTest(t, `"123"`, v)

	// list
	l := WithList()
	l.Append(WithInt(1), WithFloat(1.12312), WithBool(true))
	l2 := WithList()
	l2.Append(WithInt(2), WithString("123123"), l)
	p = NewPair("ns", "list-pair", l2)
	v = p.ToMarshalInterface()
	encodeAndTest(t, `[2,"123123",[1,1.12312,true]]`, v)

	d := WithDict()
	d.Add("di", WithInt(1231))
	d.Add("df", WithFloat(1231.1232))
	d.Add("db", WithBool(true))
	d.Add("ds", WithString("12312"))
	d.Add("d2", l)
	d.Add("d3", l2)
	p = NewPair("ns", "dict-pair", d)
	v = p.ToMarshalInterface()
	encodeAndTest(t, `{"d2":[1,1.12312,true],"d3":[2,"123123",[1,1.12312,true]],"db":true,"df":1231.1232,
"di":1231,"ds":"12312"}`, v)

}
