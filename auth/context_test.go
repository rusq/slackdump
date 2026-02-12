// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package auth

import (
	"context"
	"reflect"
	"testing"
)

type fakeProvider struct {
	simpleProvider
}

var fakeTestProvider = &fakeProvider{simpleProvider{Token: "test"}}

var (
	emptyContext        = context.Background()
	contextWithProvider = WithContext(context.Background(), fakeTestProvider)
)

func TestFromContext(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    Provider
		wantErr bool
	}{
		{
			"empty context",
			args{emptyContext},
			nil,
			true,
		},
		{
			"context with provider",
			args{contextWithProvider},
			&fakeProvider{simpleProvider{Token: "test"}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromContext(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithContext(t *testing.T) {
	type args struct {
		pctx context.Context
		p    Provider
	}
	tests := []struct {
		name string
		args args
		want context.Context
	}{
		{
			"fake provider",
			args{context.Background(), fakeTestProvider},
			contextWithProvider,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithContext(tt.args.pctx, tt.args.p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithContext() = %v, want %v", got, tt.want)
			}
			prov, err := FromContext(tt.want)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(prov, tt.args.p) {
				t.Errorf("Provider from context = %v, want %v", prov, tt.args.p)
			}
		})
	}
}
