package cassemadm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yeqown/cassem/api/concept"
)

type contentTypeParam concept.ContentType

func (p *contentTypeParam) UnmarshalJSON(data []byte) error {
	var numeric int32
	if err := json.Unmarshal(data, &numeric); err == nil {
		*p = contentTypeParam(numeric)
		return nil
	}

	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	value, ok := concept.ContentType_value[strings.ToUpper(name)]
	if !ok {
		return fmt.Errorf("unknown contentType %q", name)
	}

	*p = contentTypeParam(value)
	return nil
}

func (p contentTypeParam) concept() concept.ContentType {
	return concept.ContentType(p)
}
