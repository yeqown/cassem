package concept

import (
	"google.golang.org/protobuf/proto"
)

// MarshalProto marshals a protobuf message to bytes.
func MarshalProto(v proto.Message) ([]byte, error) {
	return proto.Marshal(v)
}

// UnmarshalProto unmarshals bytes to a protobuf message.
func UnmarshalProto(data []byte, v proto.Message) error {
	return proto.Unmarshal(data, v)
}

// MustMarshalProto marshals internal state whose encoding failure is fatal.
func MustMarshalProto(v proto.Message) []byte {
	data, err := proto.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// MustUnmarshalProto decodes internal state whose corruption is fatal.
func MustUnmarshalProto(data []byte, v proto.Message) {
	if err := proto.Unmarshal(data, v); err != nil {
		panic(err)
	}
}
