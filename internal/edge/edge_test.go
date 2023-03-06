// Package edge provides a limited implementation of Slack edge api necessary
// to get the data from a slack workspace.
package edge

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/rusq/slackdump/v2/auth"
)

var _ = godotenv.Load()

var (
	testToken  = os.Getenv("EDGE_TOKEN)")
	testCookie = os.Getenv("EDGE_COOKIE")
	testTeam   = os.Getenv("EDGE_TEAM")
)

func TestNew(t *testing.T) {
	if testToken == "" {
		t.Skip("test token not set")
	}

	prov, err := auth.NewValueAuth(testToken, testCookie)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := New(testTeam, prov.SlackToken(), prov.Cookies())
	if err != nil {
		t.Fatal(err)
	}
	req := UsersListRequest{
		Channels: []string{"C6NL0QQSG"},
		Filter:   "everyone AND NOT bots AND NOT apps",
		Count:    20,
	}
	resp, err := cl.Post(context.Background(), "/users/list", &req)
	if err != nil {
		t.Fatal(err)
	}
	var ur UsersListResponse
	if err := cl.ParseResponse(&ur, resp); err != nil {
		t.Fatal(err)
	}
	t.Error(ur)
}

func TestGetUsers(t *testing.T) {
	if testToken == "" {
		t.Skip("test token not set")
	}
	au, err := auth.NewValueAuth(testToken, testCookie)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := NewWithProvider(testTeam, au)
	if err != nil {
		t.Fatal(err)
	}
	ui, err := cl.GetUsers(context.Background(), []string{"U0LKLSNER", "U03K9GLS2", "U03KMNRQS"})
	if err != nil {
		t.Fatal(err)
	}
	t.Error(ui)
}
