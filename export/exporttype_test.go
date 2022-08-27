package export

import "testing"

func TestExportType_Set(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		wantE   ExportType
		wantErr bool
	}{
		{"nodownload", args{"nodownload"}, TNoDownload, false},
		{"standard", args{"standard"}, TStandard, false},
		{"mattermost", args{"mattermost"}, TMattermost, false},
		{"unknown", args{"gibberish"}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e = new(ExportType)
			if err := e.Set(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("ExportType.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if *e != tt.wantE {
				t.Errorf("ExportType mismatch: want: %s, got %s", tt.wantE, *e)
			}
		})
	}
}
