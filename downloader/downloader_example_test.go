package downloader_test

import (
	"context"
	"fmt"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3/downloader"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

func ExampleNew_basic() {
	client := slack.New("token")
	fs := fsadapter.NewDirectory("files/")

	dl := downloader.NewV1(
		client,
		fs,
	)

	f := &slack.File{}

	if n, err := dl.SaveFile(context.Background(), "some_dir", f); err != nil {
		fmt.Printf("failed to save the file: %s", err)
	} else {
		fmt.Printf("downloaded: %d bytes", n)
	}
}

func ExampleNew_advanced() {
	client := slack.New("token")

	// initialise the filesystem (files.zip archive)
	fs, err := fsadapter.NewZipFile("files.zip")
	if err != nil {
		fmt.Println("failed to initialise the file system")
		return
	}
	defer fs.Close()

	dl := downloader.NewV1(
		client,
		fs,
		downloader.RetriesV1(100), // 100 retries when rate limited
		downloader.LimiterV1(rate.NewLimiter(20, 1)), // rate limit
		downloader.WorkersV1(8),                      // number of download workers
	)

	f := &slack.File{}

	if n, err := dl.SaveFile(context.Background(), "some_dir", f); err != nil {
		fmt.Printf("failed to save the file: %s", err)
	} else {
		fmt.Printf("downloaded: %d bytes", n)
	}
}
