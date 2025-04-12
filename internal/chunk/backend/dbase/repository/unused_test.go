//go:build ignore

package repository

import (
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
)

var deflatedMsgA = []byte{0xaa, 0x86, 0x98, 0x64, 0xa5, 0xe4, 0xa8, 0xa4, 0xa3, 0x54, 0x52, 0xac, 0x64, 0xa5, 0x64, 0x68, 0x64, 0xac, 0x67, 0x62, 0x6a, 0xa6, 0xa4, 0x83, 0xe9, 0x5e, 0x2b, 0xb0, 0xd7, 0x75, 0x30, 0x42, 0x10, 0x26, 0xe, 0xf7, 0xba, 0x55, 0x35, 0x72, 0xd8, 0x59, 0x29, 0x29, 0xe9, 0xa0, 0x5, 0xad, 0x15, 0xd8, 0x11, 0x3a, 0xb0, 0x98, 0x82, 0x70, 0x1, 0x1, 0x0, 0x0, 0xff, 0xff}

func Test_marshalflate(t *testing.T) {
	type args struct {
		a any
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "marshals data",
			args: args{a: msgA},
			want: deflatedMsgA,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalflate(tt.args.a)
			if (err != nil) != tt.wantErr {
				t.Errorf("marshalflate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("marshalflate() = %#v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unmarshalflate(t *testing.T) {
	type args struct {
		data []byte
		v    any
	}
	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "decompresses data",
			args: args{data: deflatedMsgA, v: new(slack.Message)},
			want: &msgA,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := unmarshalflate(tt.args.data, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("unmarshalflate() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.args.v, tt.want)
		})
	}
}
