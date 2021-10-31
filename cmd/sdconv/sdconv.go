package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"github.com/rusq/slackdump"
)

const (
	outputTypeJSON = "json"
	outputTypeText = "text"

	typeMessages = "Messages"
	typeUsers    = "Users"
	typeChannels = "Channels"

	typeRegexp = `^{[\s\n]*"(Messages|Users|Channels)"`
)

var outputType string
var (
	token  string
	cookie string
)
var downloadFlag bool
var re *regexp.Regexp

func init() {
	flag.StringVar(&outputType, "o", outputTypeText, "output `format`")
	flag.BoolVar(&downloadFlag, "f", false, "download files using API token if SLACK_TOKEN environment variable is set")
	re = regexp.MustCompile(typeRegexp)
}

func main() {
	flag.Parse()

	if downloadFlag {
		token = os.Getenv("SLACK_TOKEN")
		if token == "" {
			log.Printf("file download requested but no token provided, skipping")
		}
		cookie = os.Getenv("COOKIE")
	}

	input := os.Stdin
	output := os.Stdout

	data, err := ioutil.ReadAll(input)
	if err != nil {
		log.Fatal(err)
	}

	dataType := re.FindSubmatch(data)
	if dataType == nil {
		log.Fatal("can't determine entity")
	}
	entity := string(dataType[1])

	var rep slackdump.Reporter

	switch entity {
	case typeMessages:
		var msgs slackdump.Messages
		err = json.Unmarshal(data, &msgs)
		if err == nil && token != "" {
			log.Print("fetching files")
			err = fetchFiles(&msgs, token, cookie)
		}
		msgs.SD.UpdateUserMap()
		rep = msgs
	case typeChannels:
		var chans slackdump.Channels
		err = json.Unmarshal(data, &chans)
		rep = chans
	case typeUsers:
		var users slackdump.Users
		err = json.Unmarshal(data, &users)
		rep = users
	}
	if err != nil {
		log.Fatal(err)
	}

	if err = rep.ToText(output); err != nil {
		log.Fatal(err)
	}

}

func fetchFiles(m *slackdump.Messages, tokenID string, cookie string) error {
	sd, err := slackdump.New(tokenID, cookie)
	if err != nil {
		return err
	}
	files := sd.GetFilesFromMessages(m)
	return files.DumpToDir(m.ChannelID)
}
