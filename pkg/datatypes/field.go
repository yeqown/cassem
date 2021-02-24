package datatypes

import (
	"encoding/json"

	"github.com/yeqown/cassem/pkg/hash"
)

type FieldTyp uint8

const (
	KV_FIELD_ FieldTyp = iota + 1
	LIST_FIELD_
	DICT_FIELD_
)

type IField interface {
	IEncoder

	Name() string

	Type() FieldTyp

	Value() interface{}
}

var (
	_ IField = kvField{}
	_ IField = listField{}
	_ IField = dictField{}
)

type kvField struct {
	name string

	kv IPair
}

func NewKVField(fieldKey string, p IPair) IField {
	if fieldKey == "" {
		fieldKey = hashFieldKey()
	}

	return kvField{
		name: fieldKey,
		kv:   p,
	}
}

func (k kvField) Name() string {
	return k.name
}

func (k kvField) Type() FieldTyp {
	return KV_FIELD_
}

func (k kvField) Value() interface{} {
	return k.kv
}

func (k kvField) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.kv)
}

func (k kvField) MarshalTOML() (text []byte, err error) {
	return nil, nil
	//buf := bytes.NewBuffer(nil)
	//pair := k.kv
	//text, err = pair.MarshalTOML()
	//if err != nil {
	//	return nil, errors.Wrap(err, "kvField failed marshal into TOML: "+pair.Key())
	//}
	//
	//switch pair.Value().Datatype() {
	//case DICT_DATATYPE_:
	//	buf.WriteString("[" + k.name + "]\n")
	//	buf.Write(text)
	//default:
	//	buf.WriteString(k.name + " = ")
	//	buf.Write(text)
	//}
	//buf.WriteString("\n")
	//
	//return buf.Bytes(), nil
}

type listField struct {
	name string

	pairs []IPair
}

func hashFieldKey() string {
	return "field" + hash.RandKey(6)
}

// FIXME(@yeqown): List should contains same type of pairs
func NewListField(fieldKey string, pairs []IPair) IField {
	if fieldKey == "" {
		// DONE(@yeqown): use hashed string to name this fieldKey
		fieldKey = hashFieldKey()
	}

	if pairs == nil {
		pairs = make([]IPair, 0, 4)
	}

	return listField{
		name:  fieldKey,
		pairs: pairs,
	}
}

func (k listField) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.pairs)
}

//var (
//	leftBracket   = []byte("[")
//	rightBracket  = []byte("]")
//	commaAndSpace = []byte(", ")
//)

func (k listField) MarshalTOML() (text []byte, err error) {
	return nil, nil
	//buf := bytes.NewBuffer(nil)
	//buf.Write(leftBracket)
	//for idx, pair := range k.pairs {
	//	text, err = pair.MarshalTOML()
	//	if err != nil {
	//		return nil, errors.Wrap(err, "listField failed marshal into TOML: "+pair.Key())
	//	}
	//
	//	switch pair.Value().Datatype() {
	//	case DICT_DATATYPE_:
	//		buf.WriteString("[" + "listFieldName" + k.name + "]\n")
	//		buf.Write(text)
	//	default:
	//		buf.Write(text)
	//	}
	//
	//	if idx+1 != len(k.pairs) {
	//		buf.Write(commaAndSpace)
	//	}
	//}
	//
	//buf.Write(rightBracket)
	//return buf.Bytes(), err
}

func (k listField) Name() string {
	return k.name
}

func (k listField) Type() FieldTyp {
	return LIST_FIELD_
}

func (k listField) Value() interface{} {
	return k.pairs
}

type dictField struct {
	name string

	pairs map[string]IPair
}

func NewDictField(fieldKey string, pairs map[string]IPair) IField {
	if fieldKey == "" {
		fieldKey = hashFieldKey()
	}

	if pairs == nil {
		pairs = make(map[string]IPair, 4)
	}

	return dictField{
		name:  fieldKey,
		pairs: pairs,
	}
}

func (k dictField) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.pairs)
}

func (k dictField) MarshalTOML() (text []byte, err error) {
	return nil, nil
	//	buf := bytes.NewBuffer(nil)
	//	for dictKey, pair := range k.pairs {
	//		text, err = pair.MarshalTOML()
	//		if err != nil {
	//			return nil, errors.Wrap(err, "dictField failed marshal into TOML: "+pair.Key())
	//		}
	//
	//		switch pair.Value().Datatype() {
	//		case DICT_DATATYPE_:
	//			//buf.WriteString("[" + "parentKey_todo" + "." + dictKey + "]\n")
	//			buf.WriteString("[" + dictKey + "]\n")
	//			buf.Write(text)
	//		default:
	//			buf.WriteString(dictKey + " = ")
	//			buf.Write(text)
	//		}
	//
	//		buf.WriteString("\n")
	//	}
	//
	//	return buf.Bytes(), nil
}

func (k dictField) Name() string {
	return k.name
}

func (k dictField) Type() FieldTyp {
	return DICT_FIELD_
}

func (k dictField) Value() interface{} {
	return k.pairs
}
