package control

import (
	"context"
	"testing"
)

func Test_conversationTransformer_mbeTransform(t *testing.T) {
	type fields struct {
		ctx  context.Context
		ts   ExportTransformer
		refs ReferenceChecker
	}
	type args struct {
		ctx       context.Context
		channelID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct := &conversationTransformer{
				ctx:  tt.fields.ctx,
				tf:   tt.fields.ts,
				rc: tt.fields.refs,
			}
			if err := ct.mbeTransform(tt.args.ctx, tt.args.channelID); (err != nil) != tt.wantErr {
				t.Errorf("conversationTransformer.mbeTransform() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
