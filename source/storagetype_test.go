package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageType_Set(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		e       *StorageType
		args    args
		want    StorageType
		wantErr bool
	}{
		{
			name: "STmattermost",
			e:    new(StorageType),
			args: args{v: "mattermost"},
			want: STmattermost,
		},
		{
			name: "STstandard",
			e:    new(StorageType),
			args: args{v: "standard"},
			want: STstandard,
		},
		{
			name: "STdump",
			e:    new(StorageType),
			args: args{v: "dump"},
			want: STdump,
		},
		{
			name:    "invalid",
			e:       new(StorageType),
			args:    args{v: "invalid"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.e.Set(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("StorageType.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, *tt.e)
		})
	}
}
