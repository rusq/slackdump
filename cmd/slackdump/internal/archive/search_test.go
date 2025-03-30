package archive

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_stopFn_Stop(t *testing.T) {
	noError := func() error { return nil }
	yesError := func() error { return assert.AnError }
	tests := []struct {
		name    string
		s       stopFn
		wantErr bool
	}{
		{
			name: "no error",
			s: stopFn{
				noError,
				noError,
				noError,
			},
			wantErr: false,
		},
		{
			name: "yes error",
			s: stopFn{
				noError,
				noError,
				yesError,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.Stop(); (err != nil) != tt.wantErr {
				t.Errorf("stopFn.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
