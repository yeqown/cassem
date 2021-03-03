package datatypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_WithInt(t *testing.T) {
	d := WithInt(123123)

	assert.Equal(t, INT_DATATYPE_, d.Datatype())
	assert.EqualValues(t, 123123, d.Data())
}

func Test_WithFloat(t *testing.T) {
	d := WithFloat(123.123)

	assert.Equal(t, FLOAT_DATATYPE_, d.Datatype())
	assert.EqualValues(t, 123.123, d.Data())
}

func Test_WithBool(t *testing.T) {
	d := WithBool(true)

	assert.Equal(t, BOOL_DATATYPE_, d.Datatype())
	assert.EqualValues(t, true, d.Data())
}

func Test_WithString(t *testing.T) {
	d := WithString("123123")

	assert.Equal(t, STRING_DATATYPE_, d.Datatype())
	assert.EqualValues(t, "123123", d.Data())
}

func Test_WithList(t *testing.T) {
	d := WithList()
	d.Append(WithInt(123), WithString("222"), WithBool(false), WithFloat(64.23))

	assert.Equal(t, LIST_DATATYPE_, d.Datatype())

	v := d.Data()
	ds, ok := v.(ListDT)
	require.Equal(t, true, ok)
	assert.EqualValues(t, 123, ds[0].Data())
	assert.EqualValues(t, "222", ds[1].Data())
	assert.EqualValues(t, false, ds[2].Data())
	assert.EqualValues(t, 64.23, ds[3].Data())
}

func Test_WithDict(t *testing.T) {
	d := WithDict()

	d.Add("int", WithInt(123))
	d.Add("bool", WithBool(false))
	d.Add("float", WithFloat(123.222))
	d.Add("string", WithString("123"))

	assert.Equal(t, DICT_DATATYPE_, d.Datatype())

	v := d.Data()
	ds, ok := v.(DictDT)
	require.Equal(t, true, ok)
	assert.EqualValues(t, 123, ds["int"].Data())
	assert.EqualValues(t, "123", ds["string"].Data())
	assert.EqualValues(t, false, ds["bool"].Data())
	assert.EqualValues(t, 123.222, ds["float"].Data())

}

func Test_Construct_KV(t *testing.T) {
	var i interface{} = 123
	d := FromInterface(i)
	assert.Equal(t, INT_DATATYPE_, d.Datatype())
	assert.EqualValues(t, 123, d.Data())

	var b interface{} = false
	d = FromInterface(b)
	assert.Equal(t, BOOL_DATATYPE_, d.Datatype())
	assert.EqualValues(t, false, d.Data())

	var s interface{} = "123"
	d = FromInterface(s)
	assert.Equal(t, STRING_DATATYPE_, d.Datatype())
	assert.EqualValues(t, "123", d.Data())

	var f interface{} = 12.123123
	d = FromInterface(f)
	assert.Equal(t, FLOAT_DATATYPE_, d.Datatype())
	assert.EqualValues(t, 12.123123, d.Data())
}

func Test_Construct_List(t *testing.T) {
	var l interface{} = []interface{}{1, 2, 3, 4, 5, 56}
	d := FromInterface(l)
	assert.Equal(t, LIST_DATATYPE_, d.Datatype())
	assert.EqualValues(t, ListDT{
		WithInt(1),
		WithInt(2),
		WithInt(3),
		WithInt(4),
		WithInt(5),
		WithInt(56),
	}, d.Data())
}

func Test_Construct_Dict(t *testing.T) {
	var dd interface{} = map[string]interface{}{
		"i": 1,
		"b": true,
		"f": 12312.123123,
		"s": "string",
	}
	d := FromInterface(dd)
	assert.Equal(t, DICT_DATATYPE_, d.Datatype())
	assert.EqualValues(t, DictDT{
		"b": WithBool(true),
		"f": WithFloat(12312.123123),
		"i": WithInt(1),
		"s": WithString("string"),
	}, d.Data())
}
