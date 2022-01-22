package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/rusq/slackdump/internal/tracer"

	"github.com/joho/godotenv"
	"github.com/rusq/dlog"
	"github.com/rusq/slackdump"
)

const (
	outputTypeJSON = "json"
	outputTypeText = "text"

	slackTokenEnv  = "SLACK_TOKEN"
	slackCookieEnv = "COOKIE"
)

var build = "dev"

// secrets defines the names of the supported secret files that we load our
// secrets from.  Inexperienced windows users might have bad experience trying
// to create .env file with the notepad as it will battle for having the
// "txt" extension.  Let it have it.
var secrets = []string{".env", ".env.txt", "secrets.txt"}

// params is the command line parameters
type params struct {
	creds slackCreds
	list  listFlags

	output output

	channelsToExport []string
	dumpFiles        bool

	traceFile string // trace file
	version   bool
}

type output struct {
	filename string
	format   string
}

func (out output) validFormat() bool {
	return out.format != "" && (out.format == outputTypeJSON ||
		out.format == outputTypeText)
}

type slackCreds struct {
	token  string
	cookie string
}

func (c slackCreds) valid() bool {
	return c.token != "" && c.cookie != ""
}

type listFlags struct {
	users    bool
	channels bool
}

func (lf listFlags) present() bool {
	return lf.users || lf.channels
}

func main() {
	loadSecrets(secrets)

	params, err := parseCmdLine(os.Args[1:])
	if err != nil {
		dlog.Fatal(err)
	}
	if params.version {
		fmt.Println(build)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, params); err != nil {
		dlog.Fatal(err)
	}
}

// run runs the dumper.
func run(ctx context.Context, p params) error {
	if p.traceFile != "" {
		dlog.Println("enabling trace, will write to %q", p.traceFile)
		trc := tracer.New(p.traceFile)
		if err := trc.Start(); err != nil {
			return err
		}
		defer func() {
			if err := trc.End(); err != nil {
				dlog.Printf("failed to write the trace file: %s", err)
			}
		}()
	}

	if p.list.channels || p.list.users {
		if err := listEntities(ctx, p.output, p.creds, p.list); err != nil {
			return err
		}
	} else if len(p.channelsToExport) > 0 {
		n, err := dumpChannels(ctx, p.creds, p.channelsToExport, p.dumpFiles, p.output.format == outputTypeText)
		if err != nil {
			return err
		}
		dlog.Printf("job finished, dumped %d channels", n)
	} else {
		return errors.New("nothing to do")
	}
	return nil
}

// loadSecrets load secrets from the files in secrets slice.
func loadSecrets(files []string) {
	for _, f := range files {
		godotenv.Load(f)
	}
}

// createFile creates the file, or opens the Stdout, if the filename is "-".
// It will return an error, if the things go pear-shaped.
func createFile(filename string) (f io.WriteCloser, err error) {
	if filename == "-" {
		f = os.Stdout
		return
	}
	return os.Create(filename)
}

// parseCmdLine parses the command line arguments.
func parseCmdLine(args []string) (params, error) {
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"Slackdump, %s"+
				"Slackdump dumps messages and files from slack using the provided api token.\n"+
				"Will create a number of files having the channel_id as a name.\n"+
				"Files are downloaded into a respective folder with channel_id name\n\n"+
				"Usage:  %s [flags] [channel_id1 ... channel_idN]\n\n",
			build, filepath.Base(os.Args[0]))
		fs.PrintDefaults()
	}

	var p params
	fs.BoolVar(&p.list.channels, "c", false, "list channels (aka conversations) and their IDs for export.")
	fs.BoolVar(&p.list.users, "u", false, "list users and their IDs. ")
	fs.BoolVar(&p.dumpFiles, "f", false, "enable files download")
	fs.StringVar(&p.output.filename, "o", "-", "output `filename` for users and channels.  Use '-' for standard\noutput.")
	fs.StringVar(&p.output.format, "r", "", "report `format`.  One of 'json' or 'text'")
	fs.StringVar(&p.creds.token, "t", os.Getenv(slackTokenEnv), "Specify slack `API_token`, (environment: "+slackTokenEnv+")")
	fs.StringVar(&p.creds.cookie, "cookie", os.Getenv(slackCookieEnv), "d= cookie `value` (environment: "+slackCookieEnv+")")
	fs.StringVar(&p.traceFile, "trace", os.Getenv("TRACE_FILE"), "trace `file` (optional)")
	fs.BoolVar(&p.version, "V", false, "print version and exit")

	os.Unsetenv(slackTokenEnv)
	os.Unsetenv(slackCookieEnv)

	if err := fs.Parse(args); err != nil {
		return p, err
	}

	p.channelsToExport = fs.Args()

	return p, p.validate()
}

// validate checks if the command line parameters have valid values.
func (p *params) validate() error {
	if p.version {
		return nil
	}

	if !p.creds.valid() {
		return fmt.Errorf("slack token or cookie not specified")
	}

	if len(p.channelsToExport) == 0 && !p.list.present() {
		return fmt.Errorf("no list flags specified and no channels to export")
	}
	p.creds.cookie = strings.TrimPrefix(p.creds.cookie, "d=")

	// channels and users listings will be in the text format (if not specified otherwise)
	if p.output.format == "" {
		if p.list.present() {
			p.output.format = outputTypeText
		} else {
			p.output.format = outputTypeJSON
		}
	}

	if !p.list.present() && !p.output.validFormat() {
		return fmt.Errorf("invalid output type: %q, must use one of %v", p.output.format, []string{outputTypeJSON, outputTypeText})
	}

	return nil
}

// listEntities queries lists the supported entities, and writes the output to output.
func listEntities(ctx context.Context, output output, creds slackCreds, list listFlags) error {
	w, err := createFile(output.filename)
	if err != nil {
		return err
	}
	defer w.Close()

	dlog.Print("initializing...")
	sd, err := slackdump.New(ctx, creds.token, creds.cookie)
	if err != nil {
		return err
	}

	dlog.Print("retrieving data...")

	var rep slackdump.Reporter
	switch {
	case list.channels:
		rep, err = sd.GetChannels(context.Background())
		if err != nil {
			return err
		}
	case list.users:
		rep = sd.Users
	default:
		return fmt.Errorf("don't know what to do")
	}

	dlog.Print("done")
	switch output.format {
	case outputTypeText:
		return rep.ToText(w)
	case outputTypeJSON:
		enc := json.NewEncoder(w)
		return enc.Encode(rep)
	default:
		return errors.New("invalid output format")
	}
	// unreachable
}

// dumpChannels dumps the channels with ids, if dumpfiles is true, it will save
// the files into a respective directory with ID of the channel as the name.  If
// generateText is true, it will additionally format the conversation as text
// file and write it to <ID>.txt file.
//
// The result of the work of this function, for each channel ID, the following
// files will be created:
//
//    +-<ID> - directory, if dumpfiles is true
//    |  +- attachment1.ext
//    |  +- attachment2.ext
//    |  +- ...
//    +--<ID>.json - json file with conversation and users
//    +--<ID>.txt  - formatted conversation in text format, if generateText is true.
//
func dumpChannels(ctx context.Context, creds slackCreds, ids []string, dumpfiles bool, generateText bool) (int, error) {
	sd, err := slackdump.New(ctx, creds.token, creds.cookie, slackdump.DumpFiles(dumpfiles))
	if err != nil {
		return 0, err
	}

	var total int
	for _, ch := range ids {
		dlog.Printf("dumping channel: %q", ch)

		if err := dumpOneChannel(ctx, sd, ch, generateText); err != nil {
			dlog.Printf("channel %q: %s", ch, err)
			continue
		}

		total++
	}
	return total, nil
}

// dumpOneChannel dumps just one channel having ID = id.  If generateText is
// true, it will also generate a ID.txt text file.
func dumpOneChannel(ctx context.Context, sd *slackdump.SlackDumper, id string, generateText bool) error {
	f, err := os.Create(id + ".json")
	if err != nil {
		return err
	}
	defer f.Close()

	m, err := sd.DumpMessages(ctx, id)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return err
	}
	if generateText {
		if err := formatTextFile(sd, m, id); err != nil {
			dlog.Printf("error creating text file: %s", err)
		}
	}

	return nil
}

func formatTextFile(sd *slackdump.SlackDumper, m *slackdump.Channel, id string) error {
	dlog.Printf("generating %s.txt", id)
	t, err := os.Create(id + ".txt")
	if err != nil {
		return err
	}
	defer t.Close()

	return m.ToText(sd, t)
}
