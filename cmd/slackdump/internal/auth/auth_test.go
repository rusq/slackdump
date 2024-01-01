package auth

import "testing"

func Test_argsWorkspace(t *testing.T) {
	type args struct {
		args       []string
		defaultWsp string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"empty",
			args{[]string{}, ""},
			"",
		},
		{
			"default is set, no workspace in args",
			args{[]string{}, "default"},
			"default",
		},
		{
			"default overrides args args",
			args{[]string{"arg"}, "default"},
			"default",
		},
		{
			"returns must be lowercase",
			args{[]string{"UPPERCASE"}, "DEFAULT"},
			"default",
		},
		{
			"returns must be lowercase",
			args{[]string{"UPPERCASE"}, ""},
			"uppercase",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := argsWorkspace(tt.args.args, tt.args.defaultWsp); got != tt.want {
				t.Errorf("argsWorkspace() = %v, want %v", got, tt.want)
			}
		})
	}
}
