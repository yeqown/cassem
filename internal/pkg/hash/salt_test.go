package hash

import "testing"

func TestWithSalt(t *testing.T) {
	type args struct {
		password string
		salt     string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case 0",
			args: args{
				password: "123456",
				salt:     "cassem",
			},
			want: "92f9ce613443bfa68e8d511ed579d0e29fe69778de19ab4dda10a35360940882",
		},
		{
			name: "case 1",
			args: args{
				password: "messac",
				salt:     "Y2Fzc2VuCg==",
			},
			want: "7c46f88749d0b4f39c0b089e67553361846cf9a0fa0213012ce345a5cfcea689",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithSalt(tt.args.password, tt.args.salt); got != tt.want {
				t.Errorf("WithSalt() = %v, want %v", got, tt.want)
			}
		})
	}
}
