package datatypes

import "encoding/json"

// IEncoder gather all needed serialization methods to restraint how all type in cassem acts.
type IEncoder interface {
	// json.Marshaler used to render frontend data.
	json.Marshaler

	// used to convert data into JSON format file.
	// ToJSON() ([]byte, error)

	// used to convert data into TOML format file.
	// ToTOML() ([]byte, error)
}
