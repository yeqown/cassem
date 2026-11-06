package adm

import (
	"encoding/json"

	"github.com/yeqown/cassem/internal/concept"
	"google.golang.org/protobuf/encoding/protojson"
)

// elementJSON preserves the Admin HTTP protobuf JSON contract while keeping
// serialization methods out of generated message packages.
type elementJSON struct {
	element *concept.Element
}

func (v elementJSON) MarshalJSON() ([]byte, error) {
	if v.element == nil {
		return []byte("null"), nil
	}
	return (protojson.MarshalOptions{EmitDefaultValues: true}).Marshal(v.element)
}

type elementsJSON struct {
	result *concept.GetElementsResult
}

func (v elementsJSON) MarshalJSON() ([]byte, error) {
	if v.result == nil {
		return []byte("null"), nil
	}
	var elements []elementJSON
	if v.result.Elements != nil {
		elements = make([]elementJSON, len(v.result.Elements))
		for i, element := range v.result.Elements {
			elements[i] = elementJSON{element: element}
		}
	}
	return json.Marshal(struct {
		concept.CommonPager
		Elements []elementJSON `json:"elements"`
	}{CommonPager: v.result.CommonPager, Elements: elements})
}
