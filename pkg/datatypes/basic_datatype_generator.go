package datatypes

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/yeqown/log"
)

func WithEmpty() NonData {
	return struct{}{}
}

func WithInt(i int) IntDT {
	return IntDT(i)
}

func WithFloat(f float64) FloatDT {
	return FloatDT(f)
}

func WithString(s string) StringDT {
	return StringDT(s)
}

func WithBool(b bool) BoolDT {
	return BoolDT(b)
}

// WithList returns an empty list contains nothing.
func WithList() ListDT {
	return ListDT{}
}

func FromSliceInterfaceToList(v []interface{}) ListDT {
	if v == nil {
		return nil
	}

	l := WithList()
	if len(v) == 0 {
		return l
	}

	for _, value := range v {
		l.Append(fromInterface(value))
	}

	return l
}

// WithDict returns an empty dict contains nothing.
func WithDict() DictDT {
	d := make(DictDT, 4)
	return d
}

func FromMapInterfaceToDict(v map[string]interface{}) DictDT {
	if v == nil {
		return nil
	}

	d := WithDict()
	if len(v) == 0 {
		return d
	}

	for k, value := range v {
		d.Add(k, fromInterface(value))
	}
	return d
}

var (
	defaultRepresentOpt = &option{
		expectedDatatype: EMPTY_DATATYPE,
	}
)

type option struct {
	expectedDatatype Datatype
}

type RepresentOption func(o *option)

func WithExpectedDataType(dt Datatype) func(o *option) {
	return func(o *option) {
		o.expectedDatatype = dt
	}
}

// fromInterface get the representation of v (interface{}) recursively, .
// Importantly, it panics while the value (v interface{}) could not be handled in following case:
//
// 1. v.(type) out range of integer, json.Number, float, string, bool, []interface{}, map[string]interface{}
// 2. json.Number wanted to be others Datatype excepts INT_DATATYPE_ and FLOAT_DATATYPE_.
//
// RepresentOption will help this process sometimes For example, int and float could not be distinguished
// by JSON decoder, so that we need external datatype information to help conversion handlers,
// now you can use WithExpectedDataType.
//
// Notice that json.Number is a enhancement for JSON decoder to identify the actual value. it could be interpreted
// as int64, float64 and string.
func fromInterface(v interface{}, opts ...RepresentOption) (d IData) {
	ro := defaultRepresentOpt
	for _, apply := range opts {
		apply(ro)
	}

	switch typ := v.(type) {
	case json.Number:
		var err error
		// FIXED:(@yeqown): JSON encoder can distinguish between int64 and float64 with json.Number.
		switch ro.expectedDatatype {
		case INT_DATATYPE_:
			var i int64
			i, err = v.(json.Number).Int64()
			d = WithInt(int(i))
		case FLOAT_DATATYPE_:
			var f float64
			f, err = v.(json.Number).Float64()
			d = WithFloat(f)
		default:
			err = fmt.Errorf("invalid datatype to number: %v", ro.expectedDatatype)
		}
		if err != nil {
			log.
				WithField("jsonNumber", v).
				Errorf("assertion failed to int64: %v", err)
			panic(err)
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// NOTE(@yeqown) maybe unsafe can helps this, if could not convert to int from (uint or int_x)
		d = WithInt(v.(int))
	case float64, float32:
		d = WithFloat(v.(float64))
	case string:
		d = WithString(v.(string))
	case bool:
		d = WithBool(v.(bool))
	case []interface{}:
		l := WithList()
		for _, value := range v.([]interface{}) {
			l.Append(fromInterface(value))
		}
		d = l
	case map[string]interface{}:
		l := WithDict()
		for key, value := range v.(map[string]interface{}) {
			l.Add(key, fromInterface(value))
		}
		d = l
	default:
		_ = typ
		panic(fmt.Sprintf("unsupported type: %s", reflect.TypeOf(v).String()))
	}

	return d
}

// FromInterface export fromInterface method, please look up with fromInterface to get more information.
func FromInterface(v interface{}, opts ...RepresentOption) IData {
	return fromInterface(v, opts...)
}
