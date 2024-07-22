package workspace

import (
	"bytes"
	"testing"
)

func Test_printBare(t *testing.T) {
	type args struct {
		_          manager
		current    string
		workspaces []string
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			"Test 1",
			args{
				current:    "current",
				workspaces: []string{"workspace1", "workspace2", "current"},
			},
			"workspace1\nworkspace2\n*current\n",
			false,
		},
		{
			"Test 2",
			args{
				current:    "workspace1",
				workspaces: []string{"workspace1", "workspace2", "notcurrent"},
			},
			"*workspace1\nworkspace2\nnotcurrent\n",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := printBare(w, nil, tt.args.current, tt.args.workspaces); (err != nil) != tt.wantErr {
				t.Errorf("printBare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("printBare() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
