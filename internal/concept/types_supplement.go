package concept

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/yeqown/log"

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

// convertChangeToChange convert api.Change (cassemdb.api) to concept.AgentInstanceChange.
// Make sure of that c1 is agentInstance format rather than any other.
func convertChangeToChange(c1 *apicassemdb.Change) (c2 *AgentInstanceChange, ok bool) {
	if c1 == nil {
		return
	}

	var op ChangeOp
	switch c1.GetOp() {
	case apicassemdb.Change_Set:
		op = ChangeOp_UPDATE
		if c1.GetLast() == nil {
			op = ChangeOp_NEW
		}
	case apicassemdb.Change_Unset:
		op = ChangeOp_DELETE
	default:
		return
	}

	var ins = new(AgentInstance)
	if err := UnmarshalProto(c1.GetCurrent().GetVal(), ins); err != nil {
		log.
			WithFields(log.Fields{
				"op":     op,
				"change": c1,
			}).
			Debug()
		return
	}

	// all convert OK
	ok = true
	return &AgentInstanceChange{
		Ins: ins,
		Op:  op,
	}, ok
}
