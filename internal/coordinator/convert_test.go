package coordinator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/yeqown/cassem/api/concept"
)

func marshalTestProto(t testing.TB, msg proto.Message) []byte {
	t.Helper()
	data, err := MarshalProto(msg)
	require.NoError(t, err)
	return data
}

func TestConvertFromEntitiesToElements(t *testing.T) {
	baseKey := concept.GenElementKey("app", "prod", "feature")
	metadata := &concept.ElementMetadata{Key: "feature", App: "app", Env: "prod", UsingVersion: 2}
	entity := &apikv.Entity{
		Key: concept.WithVersion(baseKey, 2),
		Val: marshalTestProto(t, &concept.Element{Version: 2, Raw: []byte("enabled"), Published: true}),
	}

	out := ConvertFromEntitiesToElements([]*apikv.Entity{
		entity,
		{Key: concept.WithVersion(baseKey, 3), Val: []byte("bad proto")},
	}, map[string]*concept.ElementMetadata{baseKey: metadata})

	require.Len(t, out, 1)
	assert.Same(t, metadata, out[0].Metadata)
	assert.Equal(t, int32(2), out[0].Version)
	assert.Equal(t, []byte("enabled"), out[0].Raw)
	assert.True(t, out[0].Published)
}

func TestConvertFromEntitiesToMetadata(t *testing.T) {
	usingKey := concept.GenElementKey("app", "prod", "using")
	unpublishedKey := concept.GenElementKey("app", "prod", "draft")
	badKey := concept.GenElementKey("app", "prod", "bad")
	using := &concept.ElementMetadata{Key: "using", App: "app", Env: "prod", UsingVersion: 2, UnpublishedVersion: 3}
	unpublished := &concept.ElementMetadata{Key: "draft", App: "app", Env: "prod", UnpublishedVersion: 4}
	entities := []*apikv.Entity{
		{Key: concept.WithMetadataSuffix(usingKey), Val: marshalTestProto(t, using)},
		{Key: concept.WithMetadataSuffix(unpublishedKey), Val: marshalTestProto(t, unpublished)},
		{Key: concept.WithMetadataSuffix(badKey), Val: []byte("bad proto")},
	}

	keys, arr, mapping := ConvertFromEntitiesToMetadata(entities, false)

	assert.Equal(t, []string{concept.WithVersion(usingKey, 2), concept.WithVersion(unpublishedKey, 4)}, keys)
	require.Len(t, arr, 2)
	assert.Equal(t, using.GetKey(), arr[0].GetKey())
	assert.Equal(t, using.GetUsingVersion(), arr[0].GetUsingVersion())
	assert.Equal(t, using.GetUnpublishedVersion(), arr[0].GetUnpublishedVersion())
	assert.Equal(t, unpublished.GetKey(), arr[1].GetKey())
	assert.Equal(t, unpublished.GetUsingVersion(), arr[1].GetUsingVersion())
	assert.Equal(t, unpublished.GetUnpublishedVersion(), arr[1].GetUnpublishedVersion())
	assert.Same(t, arr[0], mapping[usingKey])
	assert.Same(t, arr[1], mapping[unpublishedKey])
	assert.NotContains(t, mapping, badKey)

	wipedKeys, _, _ := ConvertFromEntitiesToMetadata(entities, true)
	assert.Equal(t, []string{concept.WithVersion(usingKey, 2)}, wipedKeys)
}

func TestConvertChangeToChange(t *testing.T) {
	ins := &concept.AgentInstance{AgentId: "agent-1", Addr: "127.0.0.1:9000"}
	current := &apikv.Entity{Val: marshalTestProto(t, ins)}

	tests := []struct {
		name   string
		change *apikv.Change
		wantOK bool
		wantOp concept.ChangeOp
	}{
		{name: "nil", change: nil, wantOK: false},
		{name: "set new", change: &apikv.Change{Op: apikv.Change_Set, Current: current}, wantOK: true, wantOp: concept.ChangeOp_NEW},
		{name: "set update", change: &apikv.Change{Op: apikv.Change_Set, Last: &apikv.Entity{}, Current: current}, wantOK: true, wantOp: concept.ChangeOp_UPDATE},
		{name: "unset", change: &apikv.Change{Op: apikv.Change_Unset, Current: current}, wantOK: true, wantOp: concept.ChangeOp_DELETE},
		{name: "invalid op", change: &apikv.Change{Op: apikv.Change_Invalid, Current: current}, wantOK: false},
		{name: "invalid proto", change: &apikv.Change{Op: apikv.Change_Set, Current: &apikv.Entity{Val: []byte("bad proto")}}, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ConvertChangeToChange(tt.change)

			assert.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantOp, got.Op)
			assert.Equal(t, ins.AgentId, got.Ins.AgentId)
			assert.Equal(t, ins.Addr, got.Ins.Addr)
		})
	}
}

func TestTrimVersion(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "version suffix", key: "cassem/elements/app/prod/key/v12", want: "cassem/elements/app/prod/key"},
		{name: "no version suffix", key: "cassem/elements/app/prod/key", want: "cassem/elements/app/prod/key"},
		{name: "version marker in middle", key: "cassem/elements/app/v1/key", want: "cassem/elements/app/v1/key"},
		{name: "empty", key: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, trimVersion(tt.key))
		})
	}
}

func TestTrimMetadata(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "metadata suffix", key: "cassem/elements/app/prod/key/metadata", want: "cassem/elements/app/prod/key"},
		{name: "no metadata suffix", key: "cassem/elements/app/prod/key", want: "cassem/elements/app/prod/key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, trimMetadata(tt.key))
		})
	}
}
