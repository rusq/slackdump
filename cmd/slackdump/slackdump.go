package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/joho/godotenv"
	"github.com/rusq/slackdump"
)

const (
	outputTypeJSON    = "json"
	outputTypeText    = "text"
	outputTypeDefault = ""
)

var _ = godotenv.Load()

// flags
var (
	flagChannels  = flag.Bool("c", false, "list channels/conversations and their IDs for export.  Use -ct to\nspecify channel types.")
	flagUsers     = flag.Bool("u", false, "list Users and their IDs for export. ")
	flagDumpFiles = flag.Bool("f", false, "Dump files embedded in the conversation")
	outputFile    = flag.String("o", "-", "output `filename` for users and channels.  Use '-' for standard\noutput.")
	outputType    = flag.String("r", outputTypeDefault, "report `format`.  One of 'json' or 'text'")
	tokenID       = flag.String("t", os.Getenv("SLACK_TOKEN"), "Specify slack `API_token`, get it here:\nhttps://api.slack.com/custom-integrations/legacy-tokens\n"+
		"It is also possible to define SLACK_TOKEN environment variable.")
	cookie = flag.String("cookie", os.Getenv("COOKIE"), "d= cookie value")
)

func init() {
	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(
		flag.CommandLine.Output(),
		"Slackdump dumps messages and files from slack using the provided api token.\n"+
			"Will create a number of files having the channel_id as a name.\n"+
			"Files are downloaded into a respective folder with channel_id\n\n"+
			"Usage: %s [flags] [channel_id1 ... channel_idN]\n",
		os.Args[0])
	flag.PrintDefaults()
}

func getOutputHandle(filename string) (f io.WriteCloser, err error) {
	if filename == "-" {
		f = os.Stdout
		return
	}
	return os.Create(filename)
}

func main() {
	flag.Parse()

	if err := checkParameters(); err != nil {
		flag.Usage()
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if *flagChannels || *flagUsers {
		output, err := getOutputHandle(*outputFile)
		if err != nil {
			return err
		}
		defer output.Close()

		if err = fetchData(ctx, output, *outputType); err != nil {
			return err
		}
	} else if len(flag.Args()) > 0 {
		n, err := dumpChannels(context.Background(), flag.Args())
		if err != nil {
			return err
		}
		log.Printf("job finished, dumped %d channels", n)
	} else {
		return errors.New("nothing to do")
	}
	return nil
}

func checkParameters() error {
	if *tokenID == "" {
		return fmt.Errorf("slack token not specified")
	}
	os.Unsetenv("SLACK_TOKEN")

	if *outputType != "" && !(*outputType == outputTypeJSON ||
		*outputType == outputTypeText) {
		return fmt.Errorf("invalid output type, must use one of %s or %s", outputTypeJSON, outputTypeText)
	}

	// channels and users will have a text output (if not specified otherwise)
	if *outputType == outputTypeDefault {
		if *flagChannels || *flagUsers {
			*outputType = outputTypeText
		} else {
			*outputType = outputTypeJSON
		}
	}

	if len(flag.Args()) == 0 && !(*flagChannels || *flagUsers) {
		usage()
		return fmt.Errorf("no flags specified and no channels for export")
	}

	return nil
}

func fetchData(ctx context.Context, output io.Writer, outputType string) error {
	log.Print("initializing...")
	sd, err := slackdump.New(ctx, *tokenID, *cookie)
	if err != nil {
		return err
	}

	log.Print("retrieving data...")

	var rep slackdump.Reporter
	switch {
	case *flagChannels:
		rep, err = sd.GetChannels(context.Background())
		if err != nil {
			return err
		}
	case *flagUsers:
		rep = sd.Users
	default:
		return fmt.Errorf("don't know what to do")
	}

	log.Print("done")
	switch outputType {
	case outputTypeJSON:
		data, err := json.Marshal(rep)
		if err != nil {
			return fmt.Errorf("error dumping JSON")
		}
		output.Write(data)
	case outputTypeText:
		rep.ToText(output)
	}

	return nil
}

func dumpChannels(ctx context.Context, chans []string) (int, error) {
	var n int

	sd, err := slackdump.New(ctx, *tokenID, *cookie)
	if err != nil {
		return 0, err
	}
	for _, ch := range flag.Args() {
		log.Printf("dumping channel: %q", ch)

		if err := dumpChannel(ctx, sd, ch); err != nil {
			log.Printf("channel %q: %s", ch, err)
			continue
		}

		n++
	}
	return n, nil
}

func dumpChannel(ctx context.Context, sd *slackdump.SlackDumper, c string) error {
	var wg sync.WaitGroup
	f, err := os.Create(c + ".json")
	if err != nil {
		return err
	}
	defer f.Close()

	m, err := sd.DumpMessages(ctx, c, *flagDumpFiles)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if *outputType == outputTypeText {
		wg.Add(1)
		go func() {
			log.Printf("generating %s.txt", c)
			t, err := os.Create(c + ".txt")
			if err != nil {
				log.Printf("json written ok, but text: %s", err)
			} else {
				defer t.Close()
				m.ToText(t)
			}
			wg.Done()
		}()
	}
	if err := enc.Encode(m); err != nil {
		return err
	}

	wg.Wait()
	return nil
}
