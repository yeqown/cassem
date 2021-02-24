package datatypes

import (
	"encoding/json"
)

var (
	_ IPair = builtinPair{}
)

type IPair interface {
	IEncoder

	// NS
	NS() string

	// Key
	Key() string

	// Value() IData
	Value() IData
}

// builtinPair include
type builtinPair struct {
	// namespace indicates pair would only be used in the same namespace file
	// container, and also be unique in one namespace.
	namespace string

	// key is the unique string in one namespace, usually be used to identify the builtinPair.
	key string

	// value contains basic data type
	value IData
}

func NewPair(ns, key string, value IData) IPair {
	return &builtinPair{
		namespace: ns,
		key:       key,
		value:     value,
	}
}

func (p builtinPair) NS() string {
	return p.namespace
}

func (p builtinPair) Key() string {
	return p.key
}

func (p builtinPair) Value() IData {
	return p.value
}

// FIXED: customized marshal TOML
func (p builtinPair) MarshalTOML() (text []byte, err error) {
	return nil, nil
	// p.value is basic datatype, so how to marshal as TOML.
	//return p.Value().MarshalTOML()
	//return toTomlElement(p.Value())
}

func (p builtinPair) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value)
}

//
//func toTomlElement(d IData) (text []byte, err error) {
//	var s string
//	v := d.Data()
//	switch d.Datatype() {
//	case INT_DATATYPE_, FLOAT_DATATYPE_, BOOL_DATATYPE_:
//		s = fmt.Sprintf("%v", v)
//	case STRING_DATATYPE_:
//		s = fmt.Sprintf(`"%v"`, v)
//	case LIST_DATATYPE_:
//		for _, v := range v.(ListDT) {
//			tt, _ := toTomlElement(v)
//			s += string(tt) + ", "
//		}
//		s = "[" + strings.TrimRight(s, ", ") + "]"
//	case DICT_DATATYPE_:
//		buf := bytes.NewBuffer(nil)
//		err = toml.NewEncoder(buf).Encode(v)
//		return buf.Bytes(), err
//	default:
//		err = errors.New("unsupported datatype")
//	}
//
//	return []byte(s), err
//}
