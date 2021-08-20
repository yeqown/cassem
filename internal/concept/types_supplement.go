package concept

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/runtime"
)

var _marshaler = jsonpb.Marshaler{EmitDefaults: true, EnumsAsInts: true}

func (m *Element) MarshalJSON() ([]byte, error) {
	s, err := _marshaler.MarshalToString(m)
	return runtime.ToBytes(s), err
}

func (m *ElementMetadata) MarshalJSON() ([]byte, error) {
	s, err := _marshaler.MarshalToString(m)
	return runtime.ToBytes(s), err
}

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

func convertFromEntitiesToElements(in []*apicassemdb.Entity, mdMapping map[string]*ElementMetadata) (out []*Element) {
	out = make([]*Element, 0, len(in))
	for _, entity := range in {
		elt := &Element{
			Metadata: new(ElementMetadata),
			Version:  0,
			Raw:      nil,
		}
		if err := UnmarshalProto(entity.GetVal(), elt); err != nil {
			continue
		}
		k := trimVersion(entity.GetKey())
		elt.Metadata = mdMapping[k]
		out = append(out, elt)
	}

	return out
}

// convertFromEntitiesToMetadata return keys, arr, mapping
// keys contains ElementMetadata key with version: app/env/ele/v1
// arr contains ElementMetadata in slice structure
// mapping contains ElementMetadata in format: map[app/env/ele]*ElementMetadata
func convertFromEntitiesToMetadata(
	in []*apicassemdb.Entity) (keys []string, arr []*ElementMetadata, mdMapping map[string]*ElementMetadata) {

	arr = make([]*ElementMetadata, 0, len(in))
	mdMapping = make(map[string]*ElementMetadata, len(in))
	keys = make([]string, 0, len(in))
	for _, entity := range in {
		k := trimMetadata(entity.GetKey())
		md := new(ElementMetadata)
		if err := UnmarshalProto(entity.GetVal(), md); err != nil {
			continue
		}
		md.Key = extractPureKey(k)
		arr = append(arr, md)
		mdMapping[k] = md
		// If current metadata has no using version, so there is no available version
		// for the element.
		if md.UsingVersion != 0 {
			keys = append(keys, withVersion(k, int(md.UsingVersion)))
		}
	}

	return keys, arr, mdMapping
}

// TODO(@yeqown): finish this function
func convertChangeToChange(c1 *apicassemdb.Change) (c2 *AgentInstanceChange, ok bool) {
	return &AgentInstanceChange{
		Ins: nil,
		Op:  0,
	}, ok
}
