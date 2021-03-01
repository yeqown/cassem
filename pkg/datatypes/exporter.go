package datatypes

//import (
//	"encoding/json"
//
//	"github.com/pelletier/go-toml"
//)

// IEncoder gather all needed serialization methods to restraint how all type in cassem acts.
//
// JSON specification: https://www.json.org/json-en.html
// TOML specification: https://toml.io/en/
// YAML specification: https://yaml.org/spec/1.2/spec.html
//
type IEncoder interface {
	ToMarshalInterface() interface{}
}

// ContainerFormat defines all format type of container could be encoded into.
type ContainerFormat string

func (c ContainerFormat) String() string {
	return string(c)
}

const (
	JSON ContainerFormat = "json"
	TOML ContainerFormat = "toml"
)
