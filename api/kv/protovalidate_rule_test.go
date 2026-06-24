package kv

import (
	"strings"
	"testing"

	"buf.build/go/protovalidate"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestProtovalidateKVRules(t *testing.T) {
	tests := []struct {
		name string
		msg  proto.Message
	}{
		{
			name: "get kv key must contain slash",
			msg:  &GetKVReq{Key: "noslash"},
		},
		{
			name: "get kvs keys must be unique",
			msg:  &GetKVsReq{Keys: []string{"cassem/a", "cassem/a"}},
		},
		{
			name: "set value must not exceed 256KiB",
			msg:  &SetKVReq{Key: "cassem/a", Val: []byte(strings.Repeat("x", 262145))},
		},
		{
			name: "range limit must be positive",
			msg:  &RangeReq{Key: "cassem/a", Limit: 0},
		},
		{
			name: "compact keep version count must be positive",
			msg:  &CompactElementHistoryReq{ElementKey: "cassem/elements/app/env/key", KeepVersionCount: 0, PageSize: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, protovalidate.Validate(tt.msg))
		})
	}
}

func TestProtovalidateKVRulesAcceptValidMessages(t *testing.T) {
	tests := []struct {
		name string
		msg  proto.Message
	}{
		{
			name: "valid get kv",
			msg:  &GetKVReq{Key: "cassem/a"},
		},
		{
			name: "valid get kvs",
			msg:  &GetKVsReq{Keys: []string{"cassem/a", "cassem/b"}},
		},
		{
			name: "valid set kv",
			msg:  &SetKVReq{Key: "cassem/a", Val: []byte("value")},
		},
		{
			name: "valid range",
			msg:  &RangeReq{Key: "cassem/a", Limit: 10},
		},
		{
			name: "valid compact request",
			msg: &CompactElementHistoryReq{
				ElementKey:           "cassem/elements/app/env/key",
				KeepVersionCount:     1,
				KeepVersionSeconds:   0,
				KeepOperationSeconds: 0,
				PageSize:             10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, protovalidate.Validate(tt.msg))
		})
	}
}
