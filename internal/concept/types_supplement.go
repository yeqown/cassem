package concept

import (
	"github.com/golang/protobuf/proto"
)

func (m *Instance) Id() string {
	if m.ClientId == "" {
		return "cassem" + "@" + m.GetClientIp()
	}

	return m.GetClientId() + "@" + m.GetClientIp()
}

func MarshalProto(v proto.Message) ([]byte, error) {
	return proto.Marshal(v)
}

func UnmarshalProto(data []byte, v proto.Message) error {
	return proto.Unmarshal(data, v)
}
