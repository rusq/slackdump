package export

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
)

func Test_newFileExporter(t *testing.T) {
	type args struct {
		t     ExportType
		fs    fsadapter.FS
		cl    *slack.Client
		l     *slog.Logger
		token string
	}
	tests := []struct {
		name  string
		args  args
		wantT string
	}{
		{"unknown is nodownload", args{t: ExportType(255), l: slog.Default(), token: "abcd"}, "dl.Nothing"},
		{"no", args{t: TNoDownload, l: slog.Default(), token: "abcd"}, "dl.Nothing"},
		{"standard", args{t: TStandard, fs: fsadapter.NewDirectory("."), cl: &slack.Client{}, l: slog.Default(), token: "abcd"}, "*dl.Std"},
		{"mattermost", args{t: TMattermost, fs: fsadapter.NewDirectory("."), cl: &slack.Client{}, l: slog.Default(), token: "abcd"}, "*dl.Mattermost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fe := newV2FileExporter(tt.args.t, tt.args.fs, tt.args.cl, tt.args.l, tt.args.token)
			stype := fmt.Sprintf("%T", fe)
			if stype != tt.wantT {
				t.Errorf("typeof(newFileExporter()) = %s, want %s", stype, tt.wantT)
			}
		})
	}
}
