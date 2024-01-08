package auth_ui

import "testing"

func Test_valSixDigits(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"empty",
			args{""},
			true,
		},
		{
			"too short",
			args{"12345"},
			true,
		},
		{
			"too long",
			args{"1234567"},
			true,
		},
		{
			"not a number",
			args{"123456a"},
			true,
		},
		{
			"valid",
			args{"123456"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := valSixDigits(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("valSixDigits() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
