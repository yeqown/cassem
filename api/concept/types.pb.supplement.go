package concept

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var _marshaler = protojson.MarshalOptions{EmitDefaultValues: true}

// MarshalJSON implements json.Marshaler for Element.
func (m *Element) MarshalJSON() ([]byte, error) {
	b, err := _marshaler.Marshal(m)
	return b, err
}

// MarshalJSON implements json.Marshaler for ElementMetadata.
func (m *ElementMetadata) MarshalJSON() ([]byte, error) {
	b, err := _marshaler.Marshal(m)
	return b, err
}

// Id returns a unique identifier for the Instance.
func (m *Instance) Id() string {
	if m.ClientId == "" {
		return "cassem" + "@" + m.GetClientIp()
	}

	return m.GetClientId() + "@" + m.GetClientIp()
}

// MarshalProto marshals a protobuf message to bytes.
func MarshalProto(v proto.Message) ([]byte, error) {
	return proto.Marshal(v)
}

// UnmarshalProto unmarshals bytes to a protobuf message.
func UnmarshalProto(data []byte, v proto.Message) error {
	return proto.Unmarshal(data, v)
}
