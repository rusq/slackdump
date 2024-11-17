package auth

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rusq/slackdump/v3/internal/structures"
)

func parseDotEnv(fsys fs.FS, filename string) (string, string, error) {
	const (
		tokenKey  = "SLACK_TOKEN"
		cookieKey = "SLACK_COOKIE"

		clientTokenPrefix = "xoxc-"
	)
	f, err := fsys.Open(filename)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	secrets, err := godotenv.Parse(f)
	if err != nil {
		return "", "", errors.New("not a secrets file")
	}
	token, ok := secrets[tokenKey]
	if !ok {
		return "", "", errors.New("no SLACK_TOKEN found in the file")
	}
	if err := structures.ValidateToken(token); err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(token, clientTokenPrefix) {
		return token, "", nil
	}
	cook, ok := secrets[cookieKey]
	if !ok {
		return "", "", errors.New("no SLACK_COOKIE found in the file")
	}
	if !strings.HasPrefix(cook, "xoxd-") {
		return "", "", errors.New("invalid cookie")
	}
	return token, cook, nil
}

func ParseDotEnv(filename string) (string, string, error) {
	dir := filepath.Dir(filename)
	dirfs := os.DirFS(dir)
	pth := filepath.Base(filename)
	return parseDotEnv(dirfs, pth)
}
