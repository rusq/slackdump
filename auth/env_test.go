package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rusq/slackdump/v3/internal/fixtures"
)

func mkEnvFileData(m map[string]string) []byte {
	var data []byte
	for k, v := range m {
		data = append(data, []byte(k+"="+v+"\n")...)
	}
	return data
}

func writeEnvFile(t *testing.T, filename string, m map[string]string) string {
	t.Helper()
	data := mkEnvFileData(m)
	err := os.WriteFile(filename, data, 0644)
	if err != nil {
		t.Fatal(err)
	}
	return filename
}

func Test_ParseDotEnv(t *testing.T) {
	dir := t.TempDir()
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "valid client token and cookie",
			args: args{filename: writeEnvFile(t, filepath.Join(dir, "secrets.txt"), map[string]string{
				"SLACK_TOKEN":  fixtures.TestClientToken,
				"SLACK_COOKIE": "xoxd-cookie",
			})},
			want:    fixtures.TestClientToken,
			want1:   "xoxd-cookie",
			wantErr: false,
		},
		{
			name: "valid client token but no cookie (cookie is required)",
			args: args{filename: writeEnvFile(t, filepath.Join(dir, "secrets2.txt"), map[string]string{
				"SLACK_TOKEN": fixtures.TestClientToken,
			})},
			want:    "",
			want1:   "",
			wantErr: true,
		},
		{
			name: "bot token",
			args: args{filename: writeEnvFile(t, filepath.Join(dir, "secrets3.txt"), map[string]string{
				"SLACK_TOKEN": fixtures.TestBotToken,
			})},
			want:    fixtures.TestBotToken,
			want1:   "",
			wantErr: false,
		},
		{
			name: "app token",
			args: args{filename: writeEnvFile(t, filepath.Join(dir, "secrets4.txt"), map[string]string{
				"SLACK_TOKEN": fixtures.TestAppToken,
			})},
			want:    fixtures.TestAppToken,
			want1:   "",
			wantErr: false,
		},
		{
			name: "export token",
			args: args{filename: writeEnvFile(t, filepath.Join(dir, "secrets5.txt"), map[string]string{
				"SLACK_TOKEN": fixtures.TestExportToken,
			})},
			want:    fixtures.TestExportToken,
			want1:   "",
			wantErr: false,
		},
		{
			name: "legacy token",
			args: args{filename: writeEnvFile(t, filepath.Join(dir, "secrets6.txt"), map[string]string{
				"SLACK_TOKEN": fixtures.TestPersonalToken,
			})},
			want:    fixtures.TestPersonalToken,
			want1:   "",
			wantErr: false,
		},
		{
			name: "invalid token",
			args: args{filename: writeEnvFile(t, filepath.Join(dir, "secrets7.txt"), map[string]string{
				"SLACK_TOKEN": "invalid",
			})},
			want:    "",
			want1:   "",
			wantErr: true,
		},
		{
			name: "missing token",
			args: args{filename: writeEnvFile(t, filepath.Join(dir, "secrets8.txt"), map[string]string{
				"NOT_SLACK_TOKEN": "invalid",
			})},
			want:    "",
			want1:   "",
			wantErr: true,
		},
		{
			name:    "non-existent file",
			args:    args{filename: filepath.Join(dir, "secrets9.txt")},
			want:    "",
			want1:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ParseDotEnv(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDotEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseDotEnv() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ParseDotEnv() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
