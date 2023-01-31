package export

import (
	"testing"
	"time"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

func TestOptions_IsFilesEnabled(t *testing.T) {
	type fields struct {
		Oldest      time.Time
		Latest      time.Time
		Logger      logger.Interface
		List        *structures.EntityList
		Type        ExportType
		ExportToken string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"files disabled", fields{Type: TNoDownload}, false},
		{"files enabled (standard)", fields{Type: TStandard}, true},
		{"files enabled (mattermost)", fields{Type: TMattermost}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := Options{
				Oldest:      tt.fields.Oldest,
				Latest:      tt.fields.Latest,
				Logger:      tt.fields.Logger,
				List:        tt.fields.List,
				Type:        tt.fields.Type,
				ExportToken: tt.fields.ExportToken,
			}
			if got := opt.IsFilesEnabled(); got != tt.want {
				t.Errorf("Options.IsFilesEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
