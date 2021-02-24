package datatypes

// Datatype indicates to which basic datatype the IData belongs.
type Datatype uint8

const (
	// EMPTY_DATATYPE help construct a IPair with
	EMPTY_DATATYPE Datatype = iota
	// basic datatype
	INT_DATATYPE_
	STRING_DATATYPE_
	FLOAT_DATATYPE_
	BOOL_DATATYPE_
	// container datatype
	LIST_DATATYPE_
	DICT_DATATYPE_
)

var (
	_ IData = FloatDT(12312.123123)
	_ IData = StringDT("12312.123123")
	_ IData = IntDT(12312)
	_ IData = BoolDT(true)
	_ IData = ListDT{}
	_ IData = DictDT{}
)

// IData contains basic data types
type IData interface {
	// Datatype to indicates the datatype of data.
	Datatype() Datatype

	// Value returns readonly value of data.
	Data() interface{}
}

type NonData struct{}

func (n NonData) Datatype() Datatype {
	return EMPTY_DATATYPE
}

func (n NonData) Data() interface{} {
	return nil
}

type FloatDT float64

func (f FloatDT) Datatype() Datatype {
	return FLOAT_DATATYPE_
}

func (f FloatDT) Data() interface{} {
	return f
}

type IntDT int64

func (i IntDT) Datatype() Datatype {
	return INT_DATATYPE_
}

func (i IntDT) Data() interface{} {
	return i
}

type StringDT string

func (s StringDT) Datatype() Datatype {
	return STRING_DATATYPE_
}

func (s StringDT) Data() interface{} {
	return s
}

type BoolDT bool

func (b BoolDT) Datatype() Datatype {
	return BOOL_DATATYPE_
}

func (b BoolDT) Data() interface{} {
	return b
}

type ListDT []IData

func (l ListDT) Datatype() Datatype {
	return LIST_DATATYPE_
}

func (l ListDT) Data() interface{} {
	return l
}

func (l *ListDT) Append(vs ...IData) {
	if l == nil {
		panic("ListDT is not initialized")
	}

	*l = append(*l, vs...)
}

type DictDT map[string]IData

func (d DictDT) Datatype() Datatype {
	return DICT_DATATYPE_
}

func (d DictDT) Data() interface{} {
	return d
}

func (d DictDT) Add(key string, v IData) {
	if d == nil {
		panic("DictDT is not initialized")
	}

	d[key] = v
}

func (d DictDT) Remove(key string) {
	if d == nil || len(d) == 0 {
		return
	}
	delete(d, key)
}
