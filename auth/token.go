package auth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/rusq/slackauth"
)

var ssbURI = func(workspace string) string {
	return "https://" + workspace + ".slack.com/ssb/redirect"
}

func getTokenByCookie(ctx context.Context, workspaceName string, dCookie string) (string, []*http.Cookie, error) {
	if dCookie == "" {
		return "", nil, ErrNoCookies
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ssbURI(workspaceName), nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("User-Agent", slackauth.DefaultUserAgent)
	req.Header.Add("Cookie", "d="+dCookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}
	token, err := extractToken(resp.Body)
	if err != nil {
		return "", nil, err
	}
	cookies := append(resp.Cookies(), makeCookie("d", dCookie))
	return token, cookies, nil
}

var tokenRegex = regexp.MustCompile(`"api_token":"([^"]+)"`)

var errNoToken = errors.New("token not found")

// extractToken extracts the API token from the provided reader.
// It expects that reader points to an HTML page retrieved from
// /ssb/redirect
func extractToken(r io.Reader) (string, error) {
	var token string
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return token, errNoToken
			}
			return "", fmt.Errorf("read: %w", err)
		}
		text := strings.TrimSpace(line)
		if !strings.Contains(text, "api_token") {
			continue
		}
		matches := tokenRegex.FindStringSubmatch(text)
		if len(matches) < 2 || (len(matches) == 2 && matches[1] == "") {
			return "", errNoToken
		}
		token = matches[1]
		break
	}
	return token, nil
}
