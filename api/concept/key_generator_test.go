package concept

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_trimVersion(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case 0",
			args: args{key: "a/b/c/v1"},
			want: "a/b/c",
		},
		{
			name: "case 1",
			args: args{key: "a"},
			want: "a",
		},
		{
			name: "case 2",
			args: args{key: "a/b"},
			want: "a/b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimVersion(tt.args.key); got != tt.want {
				t.Errorf("trimVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenInstanceReversedKeyUsesNonCollidingSeparator(t *testing.T) {
	first := GenInstanceReversedKey("a-b", "c", "d")
	second := GenInstanceReversedKey("a", "b-c", "d")
	if first == second {
		t.Fatalf("reverse keys should not collide: %q", first)
	}

	want := "cassem/instances/reversed/a-b@@@c@@@d"
	if first != want {
		t.Fatalf("GenInstanceReversedKey() = %q, want %q", first, want)
	}
}

func TestGenInstanceReversedKeyWithInsIdUsesNonCollidingSeparator(t *testing.T) {
	got := GenInstanceReversedKeyWithInsId("a-b", "c", "d", "ins-1")
	want := "cassem/instances/reversed/a-b@@@c@@@d/ins-1"
	if got != want {
		t.Fatalf("GenInstanceReversedKeyWithInsId() = %q, want %q", got, want)
	}
}

func TestInstanceWatchingIdentifierValidation(t *testing.T) {
	valid := &Instance_Watching{App: "demo_app", Env: "prod-1", WatchKeys: []string{"db_url", "feature_flag"}}
	require.NoError(t, valid.Validate())

	for _, watching := range []*Instance_Watching{
		{App: "demo.app", Env: "prod", WatchKeys: []string{"db_url"}},
		{App: "demo", Env: "prod env", WatchKeys: []string{"db_url"}},
		{App: "demo", Env: "prod", WatchKeys: []string{"db.url"}},
		{App: "demo", Env: "prod", WatchKeys: []string{"feature,flag"}},
	} {
		require.Error(t, watching.Validate())
	}
}

func Test_trimMetadata(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case 0",
			args: args{key: "a/b/c/v1"},
			want: "a/b/c/v1",
		},
		{
			name: "case 1",
			args: args{key: "a"},
			want: "a",
		},
		{
			name: "case 2",
			args: args{key: "a/b/metadata"},
			want: "a/b",
		},
		{
			name: "case 3",
			args: args{key: "metadata"},
			want: "metadata",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimMetadata(tt.args.key); got != tt.want {
				t.Errorf("trimMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}
