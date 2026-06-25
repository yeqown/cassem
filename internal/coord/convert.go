package coord

import (
	apikv "github.com/yeqown/cassem/api/kv"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/log"
)

// Note: The following functions were moved from api/concept/types.pb.supplement.go
// to resolve circular dependency.

// MarshalProto marshals a protobuf message to bytes.
func MarshalProto(v proto.Message) ([]byte, error) {
	return proto.Marshal(v)
}

// UnmarshalProto unmarshals bytes to a protobuf message.
func UnmarshalProto(data []byte, v proto.Message) error {
	return proto.Unmarshal(data, v)
}

// ConvertFromEntitiesToElements converts cassemkv entities to concept Elements.
func ConvertFromEntitiesToElements(in []*apikv.Entity, mdMapping map[string]*concept.ElementMetadata) (out []*concept.Element) {
	out = make([]*concept.Element, 0, len(in))
	for _, entity := range in {
		elt := &concept.Element{
			Metadata: new(concept.ElementMetadata),
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

// ConvertFromEntitiesToMetadata converts cassemkv entities to concept ElementMetadata.
// Returns:
// - keys: ElementMetadata keys with version: app/env/ele/v1
// - arr: ElementMetadata in slice structure
// - mdMapping: ElementMetadata in format: map[app/env/ele]*ElementMetadata
func ConvertFromEntitiesToMetadata(
	in []*apikv.Entity, wipeUnpublish bool,
) (keys []string, arr []*concept.ElementMetadata, mdMapping map[string]*concept.ElementMetadata) {

	arr = make([]*concept.ElementMetadata, 0, len(in))
	mdMapping = make(map[string]*concept.ElementMetadata, len(in))
	keys = make([]string, 0, len(in))
	for _, entity := range in {
		k := trimMetadata(entity.GetKey())
		md := new(concept.ElementMetadata)
		if err := UnmarshalProto(entity.GetVal(), md); err != nil {
			continue
		}
		arr = append(arr, md)
		mdMapping[k] = md
		// If current metadata has no using version, so there is no available version
		// for the element.
		if md.UsingVersion != 0 {
			keys = append(keys, concept.WithVersion(k, int(md.UsingVersion)))
			continue
		}

		if !wipeUnpublish && md.UsingVersion <= 0 && md.UnpublishedVersion != 0 {
			keys = append(keys, concept.WithVersion(k, int(md.UnpublishedVersion)))
			continue
		}
	}

	return keys, arr, mdMapping
}

// ConvertChangeToChange converts kv.Change (cassemkv.api) to concept.AgentInstanceChange.
// Make sure of that c1 is agentInstance format rather than any other.
func ConvertChangeToChange(c1 *apikv.Change) (c2 *concept.AgentInstanceChange, ok bool) {
	if c1 == nil {
		return
	}

	var op concept.ChangeOp
	switch c1.GetOp() {
	case apikv.Change_Set:
		op = concept.ChangeOp_UPDATE
		if c1.GetLast() == nil {
			op = concept.ChangeOp_NEW
		}
	case apikv.Change_Unset:
		op = concept.ChangeOp_DELETE
	default:
		return
	}

	var ins = new(concept.AgentInstance)
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
	return &concept.AgentInstanceChange{
		Ins: ins,
		Op:  op,
	}, ok
}

// trimVersion removes the version suffix from a key.
// Example: "cassem/elements/app/env/key/v1" -> "cassem/elements/app/env/key"
func trimVersion(key string) string {
	idx := strings.LastIndex(key, "/v")
	if idx <= 0 {
		return key
	}
	num := key[idx+2:]
	if _, err := strconv.Atoi(num); err != nil {
		return key
	}
	return key[:idx]
}

// trimMetadata removes the /metadata suffix from a key.
// Example: "cassem/elements/app/env/key/metadata" -> "cassem/elements/app/env/key"
func trimMetadata(key string) string {
	return strings.TrimSuffix(key, "/metadata")
}
