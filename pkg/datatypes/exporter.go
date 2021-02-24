package datatypes

import (
	"encoding"
	"encoding/json"
)

// IEncoder gather all needed serialization methods to restraint how all type in cassem acts.
type IEncoder interface {
	// json.Marshaler used to render frontend data.
	json.Marshaler

	// encoding.TextMarshaler used to convert data into TOML format file.
	encoding.TextMarshaler
}
