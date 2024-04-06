package status

import (
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/stretchr/testify/assert"
)

func TestStatus_View(t *testing.T) {
	type fields struct {
		v      viewport.Model
		params *Parameters
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "ok",
			fields: fields{
				v:      viewport.Model{Height: 2, Width: 20},
				params: NewParameters(Parameter{Name: "test", Value: "value"}),
			},
			want: "test: value         \n                    ",
		},
		{
			name: "2 params",
			fields: fields{
				v:      viewport.Model{Height: 2},
				params: NewParameters(Parameter{Name: "test", Value: "value"}, Parameter{Name: "test2", Value: true}),
			},
			want: "test:  value\ntest2: true ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				v:      tt.fields.v,
				params: tt.fields.params,
			}
			assert.Equal(t, tt.want, m.View())
		})
	}
}
