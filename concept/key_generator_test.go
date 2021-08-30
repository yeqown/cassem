package concept

import "testing"

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
