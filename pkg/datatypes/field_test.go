package datatypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_KVField(t *testing.T) {
	sp := NewPair("ns", "string", WithString("this is a string"))
	stringField := NewKVField("kv", sp)

	assert.EqualValues(t, "kv", stringField.Name())
	assert.EqualValues(t, KV_FIELD_, stringField.Type())
	assert.EqualValues(t, sp, stringField.Value())
}

func Test_KVField_TOML(t *testing.T) {
	//t.Skip("not supported yet")
	l := WithList()
	l.Append(WithInt(123), WithString("222"), WithBool(false), WithFloat(64.23))

	d := WithDict()
	d.Add("d1", WithString("222"))
	d.Add("d2", WithInt(222))

	field := NewKVField("dict", NewPair("ns", "dict", d))
	content, err := field.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, content)
	t.Logf("dict %s", content)

	field = NewKVField("list", NewPair("ns", "list", l))
	content, err = field.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, content)
	t.Logf("list: %s", content)

	field = NewKVField("list", NewPair("ns", "bool", WithBool(true)))
	content, err = field.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, content)
	t.Logf("kv: %s", content)

}

func Test_ListField(t *testing.T) {
	l := WithList()
	l.Append(WithInt(123), WithString("222"), WithBool(false), WithFloat(64.23))

	d := WithDict()
	d.Add("d1", WithString("222"))
	d.Add("d2", WithInt(222))

	pairs := []IPair{
		NewPair("ns", "int", WithInt(123)),
		NewPair("ns", "string", WithString("222")),
		NewPair("ns", "float", WithFloat(64.23)),
		NewPair("ns", "bool", WithBool(false)),
		NewPair("ns", "dict", d),
		NewPair("ns", "list", l),
	}

	field := NewListField("list", pairs)

	assert.EqualValues(t, "list", field.Name())
	assert.EqualValues(t, LIST_FIELD_, field.Type())
	assert.IsType(t, []IPair{}, field.Value())
}

func Test_ListField_TOML(t *testing.T) {
	//t.Skip("not supported yet")
	l := WithList()
	l.Append(WithInt(123), WithString("222"), WithBool(false), WithFloat(64.23))

	d := WithDict()
	d.Add("d1", WithString("222"))
	d.Add("d2", WithInt(222))

	pairs := []IPair{
		NewPair("ns", "int", WithInt(123)),
		NewPair("ns", "string", WithString("222")),
		NewPair("ns", "float", WithFloat(64.23)),
		NewPair("ns", "bool", WithBool(false)),
		NewPair("ns", "dict", d),
		NewPair("ns", "list", l),
	}

	field := NewListField("list", pairs)
	content, err := field.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, content)
	t.Logf("list-field: %s", content)
}

func Test_DictField(t *testing.T) {
	l := WithList()
	l.Append(WithInt(123), WithString("222"), WithBool(false), WithFloat(64.23))

	d := WithDict()
	d.Add("d1", WithString("222"))
	d.Add("d2", WithInt(222))

	pairs := map[string]IPair{
		"int":    NewPair("ns", "int", WithInt(123)),
		"string": NewPair("ns", "string", WithString("222")),
		"float":  NewPair("ns", "float", WithFloat(64.23)),
		"bool":   NewPair("ns", "bool", WithBool(false)),
		"dict":   NewPair("ns", "dict", d),
		"list":   NewPair("ns", "list", l),
	}

	field := NewDictField("dict", pairs)

	assert.EqualValues(t, "dict", field.Name())
	assert.EqualValues(t, DICT_FIELD_, field.Type())
	assert.IsType(t, map[string]IPair{}, field.Value())
}

func Test_DictField_TOML(t *testing.T) {
	//t.Skip("not supported yet")
	l := WithList()
	l.Append(WithInt(123), WithString("222"), WithBool(false), WithFloat(64.23))

	d := WithDict()
	d.Add("d1", WithString("222"))
	d.Add("d2", WithInt(222))

	pairs := map[string]IPair{
		"int":    NewPair("ns", "int", WithInt(123)),
		"string": NewPair("ns", "string", WithString("222")),
		"float":  NewPair("ns", "float", WithFloat(64.23)),
		"bool":   NewPair("ns", "bool", WithBool(false)),
		"dict":   NewPair("ns", "dict", d),
		"list":   NewPair("ns", "list", l),
	}

	field := NewDictField("dict", pairs)
	content, err := field.MarshalTOML()
	assert.Nil(t, err)
	assert.NotEmpty(t, content)
	t.Logf("dictField: \n%s", content)
}
