package auth

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    ValueAuth
		wantErr bool
	}{
		{
			"loads valid data",
			args{strings.NewReader(`{"Token":"token_value","Cookie":[{"Name":"d","Value":"abc","Path":"","Domain":"","Expires":"0001-01-01T00:00:00Z","RawExpires":"","MaxAge":0,"Secure":false,"HttpOnly":false,"SameSite":0,"Raw":"","Unparsed":null}]}`)},
			ValueAuth{simpleProvider{Token: "token_value", Cookie: []http.Cookie{
				{Name: "d", Value: "abc"},
			}}},
			false,
		},
		{
			"corrupt data",
			args{strings.NewReader(`{`)},
			ValueAuth{},
			true,
		},
		{
			"no data",
			args{strings.NewReader(``)},
			ValueAuth{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSave(t *testing.T) {
	type args struct {
		p Provider
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			"all info present",
			args{ValueAuth{simpleProvider{Token: "token_value", Cookie: []http.Cookie{
				{Name: "d", Value: "abc"},
			}}}},
			`{"Token":"token_value","Cookie":[{"Name":"d","Value":"abc","Path":"","Domain":"","Expires":"0001-01-01T00:00:00Z","RawExpires":"","MaxAge":0,"Secure":false,"HttpOnly":false,"SameSite":0,"Raw":"","Unparsed":null}]}` + "\n",
			false,
		},
		{
			"token missing",
			args{ValueAuth{simpleProvider{Token: "", Cookie: []http.Cookie{
				{Name: "d", Value: "abc"},
			}}}},
			"",
			true,
		},
		{
			"cookies missing",
			args{ValueAuth{simpleProvider{Token: "token_value", Cookie: []http.Cookie{}}}},
			"",
			true,
		},
		{
			"token and cookie are missing",
			args{ValueAuth{simpleProvider{}}},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := Save(w, tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotW := w.String()
			if gotW != tt.wantW {
				t.Errorf("Save() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
