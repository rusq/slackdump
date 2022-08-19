package main

import (
	"context"
	"fmt"
	"log"

	"github.com/playwright-community/playwright-go"

	"github.com/rusq/slackdump/v2/auth/browser"
)

func init() {
	_ = playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}})
}

func main() {
	b, err := browser.New("ora600")
	if err != nil {
		log.Fatal(err)
	}
	token, cookies, err := b.Authenticate(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(token)
	fmt.Println(cookies)
	fmt.Println(err)
}
