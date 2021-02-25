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

	field := NewKVField("kv-dict", NewPair("ns", "dict", d))
	v := field.ToMarshalInterface()
	encodeAndTest(t, `{"d1":"222","d2":222}`, v)

	field = NewKVField("kv-list", NewPair("ns", "list", l))
	v = field.ToMarshalInterface()
	encodeAndTest(t, `[123,"222",false,64.23]`, v)

	field = NewKVField("kv-bool", NewPair("ns", "bool", WithBool(true)))
	v = field.ToMarshalInterface()
	encodeAndTest(t, "true", v)
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
	v := field.ToMarshalInterface()
	encodeAndTest(t, `[123,"222",64.23,false,{"d1":"222","d2":222},[123,"222",false,64.23]]`, v)
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
	v := field.ToMarshalInterface()

	encodeAndTest(t, `{"bool":false,"dict":{"d1":"222","d2":222},"float":64.23,"int":123,
"list":[123,"222",false,64.23],"string":"222"}`, v)

}
