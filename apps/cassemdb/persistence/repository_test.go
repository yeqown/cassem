package persistence

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	_emptyNodes []string
	_emptyLeaf  string
)

func TestKeySplitter(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name      string
		args      args
		wantNodes []string
		wantLeaf  string
	}{
		{
			name:      "case 0",
			args:      args{s: "/a"},
			wantNodes: []string{""},
			wantLeaf:  "a",
		},
		{
			name:      "case 1",
			args:      args{s: "a/"},
			wantNodes: []string{"a"},
			wantLeaf:  _emptyLeaf,
		},
		{
			name:      "case 2",
			args:      args{s: "a/b/c/d"},
			wantNodes: []string{"a", "b", "c"},
			wantLeaf:  "d",
		},
		{
			name:      "case 3",
			args:      args{s: "/"},
			wantNodes: []string{""},
			wantLeaf:  _emptyLeaf,
		},
		{
			name:      "case 4",
			args:      args{s: "a"},
			wantNodes: _emptyNodes,
			wantLeaf:  "a",
		},
		{
			name:      "case 5",
			args:      args{s: ""},
			wantNodes: _emptyNodes,
			wantLeaf:  _emptyLeaf,
		},
		{
			name:      "case 6",
			args:      args{s: "a/b"},
			wantNodes: []string{"a"},
			wantLeaf:  "b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodes, gotLeaf := KeySplitter(tt.args.s)
			assert.Equal(t, tt.wantNodes, gotNodes)
			assert.Equal(t, tt.wantLeaf, gotLeaf)
		})
	}
}
