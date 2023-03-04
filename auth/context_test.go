package auth

import (
	"context"
	"reflect"
	"testing"
)

type fakeProvider struct {
	simpleProvider
}

func (fakeProvider) Type() Type {
	return Type(99)
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
