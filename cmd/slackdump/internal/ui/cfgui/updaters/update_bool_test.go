package updaters

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func ptr[T any](v T) *T { return &v }

func Test_boolUpdateModel_Update(t *testing.T) {
	type fields struct {
		v *bool
	}
	type args struct {
		msg tea.Msg
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   BoolModel
		want1  tea.Cmd
	}{
		{
			name: "set value message",
			fields: fields{
				v: ptr(false),
			},
			args: args{
				msg: cmdSetValue("", true)(),
			},
			want:  BoolModel{Value: ptr(true)},
			want1: OnClose,
		},
		{
			name: "unknown message",
			fields: fields{
				v: ptr(false),
			},
			args: args{
				msg: tea.Key{},
			},
			want:  BoolModel{Value: ptr(false)},
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := BoolModel{
				Value: tt.fields.v,
			}
			got, got1 := m.Update(tt.args.msg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("boolUpdateModel.Update() got = %v, want %v", got.View(), tt.want.View())
			}
			if ((tt.want1 == nil) && (got1 != nil)) || ((tt.want1 != nil) && (got1 == nil)) {
				t.Fatalf("boolUpdateModel.Update() got1 = %v, want %v", got1, tt.want1)
			} else if tt.want1 != nil && got1 != nil && !reflect.DeepEqual(got1(), tt.want1()) {
				t.Errorf("boolUpdateModel.Update() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_boolUpdateModel_Init(t *testing.T) {
	type fields struct {
		v *bool
	}
	tests := []struct {
		name   string
		fields fields
		want   tea.Cmd
	}{
		{
			name: "init should invert the stored value",
			fields: fields{
				v: ptr(false),
			},
			want: cmdSetValue("", true),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := BoolModel{
				Value: tt.fields.v,
			}
			if got := m.Init(); !reflect.DeepEqual(got(), tt.want()) {
				t.Errorf("boolUpdateModel.Init() = %v, want %v", got, tt.want)
			}
		})
	}
}
