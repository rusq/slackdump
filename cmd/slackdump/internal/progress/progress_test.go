package progress

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/progresslog"
	"github.com/rusq/slackdump/v4/stream"
)

func testReporter() *Reporter {
	return &Reporter{channels: map[string]*job{}, threads: map[key]*job{}, waits: map[uint64]time.Time{}, discovering: true, channelDiscovering: true, start: time.Now(), width: func() int { return 80 }}
}

func TestReporter_Event(t *testing.T) {
	t.Run("concurrency", eventConcurrency)
	r := testReporter()
	event := func(k stream.ProgressKind, c, ts string, err error) {
		r.Event(stream.ProgressEvent{Kind: k, ChannelID: c, ThreadTS: ts, Err: err})
	}
	// Unknown completions must not create jobs; a thread job with an empty
	// timestamp is still a thread for thread-specific events.
	event(stream.ProgressChannelDone, "missing", "", nil)
	event(stream.ProgressResultProcessed, "missing", "", errors.New("failed"))
	event(stream.ProgressThreadDone, "missing", "", nil)
	if len(r.channels) != 0 || len(r.threads) != 0 {
		t.Fatal("completion created a job")
	}
	event(stream.ProgressThreadQueued, "empty", "", nil)
	event(stream.ProgressThreadDone, "empty", "", nil)
	if j := r.threads[key{"empty", ""}]; j == nil || j.pending != 0 || r.channels["empty"] != nil {
		t.Fatal("empty thread timestamp misrouted")
	}
	delete(r.threads, key{"empty", ""})
	event(stream.ProgressChannelQueued, "C", "", nil)
	event(stream.ProgressThreadQueued, "C", "1", nil)
	event(stream.ProgressThreadQueued, "C", "1", nil)
	event(stream.ProgressThreadDone, "C", "1", nil)
	if done, _ := count(r.threads); done != 0 {
		t.Fatalf("duplicate job resolved early: %d", done)
	}
	event(stream.ProgressThreadDone, "C", "1", errors.New("failed"))
	event(stream.ProgressChannelDone, "C", "", nil)
	event(stream.ProgressInputDone, "", "", nil)
	if r.channelDiscovering || !r.discovering {
		t.Fatal("input close must freeze channel totals independently")
	}
	event(stream.ProgressDiscoveryDone, "", "", nil)
	if done, failed := count(r.threads); done != 0 || failed != 1 {
		t.Fatalf("got %d resolved and %d failed", done, failed)
	}
	if done, _ := count(r.channels); done != 1 {
		t.Fatalf("channel scans = %d", done)
	}
	event(stream.ProgressThreadQueued, "C", "2", nil)
	event(stream.ProgressResultQueued, "C", "2", nil)
	event(stream.ProgressThreadDone, "C", "2", nil)
	if done, _ := count(r.threads); done != 0 {
		t.Fatal("unprocessed callback counted complete")
	}
	event(stream.ProgressResultProcessed, "C", "2", nil)
	if done, _ := count(r.threads); done != 1 {
		t.Fatal("processed callback not resolved")
	}
	event(stream.ProgressResultProcessed, "C", "2", errors.New("callback"))
	if done, failed := count(r.threads); done != 0 || failed != 2 {
		t.Fatalf("callback failure not tracked: %d %d", done, failed)
	}
}

func eventConcurrency(t *testing.T) {
	r := testReporter()
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 50 {
				r.Event(stream.ProgressEvent{Kind: stream.ProgressThreadQueued, ChannelID: "C", ThreadTS: "T"})
				r.Event(stream.ProgressEvent{Kind: stream.ProgressThreadDone, ChannelID: "C", ThreadTS: "T"})
				_ = r.lines(80)
			}
		})
	}
	wg.Wait()
	if done, failed := count(r.threads); done != 1 || failed != 0 {
		t.Fatalf("got %d %d", done, failed)
	}
}

func Test_elapsed(t *testing.T) {
	if got := elapsed(26*time.Hour + 14*time.Minute + 32*time.Second); got != "26:14:32" {
		t.Fatal(got)
	}
}

func TestReporter_draw(t *testing.T) {
	r := testReporter()
	var out bytes.Buffer
	r.output = &out
	r.tty = false
	r.draw(false)
	if strings.Contains(out.String(), "\x1b") || strings.Count(out.String(), "\n") != 3 {
		t.Fatalf("non-tty snapshot: %q", out.String())
	}
	out.Reset()
	r.tty = true
	r.width = func() int { return 35 }
	r.draw(false)
	r.draw(false)
	if !strings.Contains(out.String(), "\x1b[3A\r\x1b[J") {
		t.Fatalf("panel was not cleared: %q", out.String())
	}
	for _, line := range strings.Split(strings.ReplaceAll(out.String(), "\x1b[3A\r\x1b[J", ""), "\n") {
		if len(line) > 34 {
			t.Fatalf("line not truncated: %q", line)
		}
	}
}

func TestHandler_Handle(t *testing.T) {
	for _, tc := range []struct {
		name              string
		separate, verbose bool
	}{
		{name: "terminal quiet"}, {name: "terminal verbose", verbose: true}, {name: "file quiet", separate: true}, {name: "file verbose", separate: true, verbose: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := testReporter()
			var terminal, file bytes.Buffer
			r.output = &terminal
			r.tty = false
			r.separateLog = tc.separate
			var next slog.Handler
			if tc.separate {
				next = slog.NewJSONHandler(&file, &slog.HandlerOptions{Level: slog.LevelDebug})
			} else {
				next = slog.NewTextHandler(&terminal, &slog.HandlerOptions{Level: slog.LevelDebug})
			}
			console := slog.NewTextHandler(&terminal, &slog.HandlerOptions{Level: slog.LevelDebug})
			logger := slog.New(&handler{next: next, console: console, r: r}).With("job", "example").WithGroup("stage").With("channel", "C123")
			logger.Debug("diagnostic")
			logger.Info("stream")
			logger.Info("resuming after rate limit")
			logger.Info("got rate limited", "source", "unmarked")
			logger.InfoContext(rateLimitLogContext(t.Context()), "got rate limited, sleeping", "source", "Slack API", "retry_after", "7s")
			logger.Warn("canvas discussions unavailable", "error", errors.New("private wrapped cause"))
			logger.Error("failed")
			got := terminal.String()
			for _, hidden := range []string{"diagnostic", "resuming after rate limit", "stream", "unmarked", "Slack API", "7s", "C123", "example", "private wrapped cause", "[2026-"} {
				if strings.Contains(got, hidden) {
					t.Fatalf("unexpected console detail %q: %q", hidden, got)
				}
			}
			if strings.Count(got, "got rate limited, sleeping") != 1 || strings.Count(got, "canvas discussions unavailable") != 1 || strings.Count(got, "failed") != 1 || strings.Count(got, "\n") != 3 {
				t.Fatalf("unexpected console records: %q", got)
			}
			if tc.separate {
				rows := strings.Split(strings.TrimSpace(file.String()), "\n")
				if len(rows) != 7 {
					t.Fatalf("file has %d records: %q", len(rows), file.String())
				}
				var records []map[string]any
				for _, row := range rows {
					var v map[string]any
					if err := json.Unmarshal([]byte(row), &v); err != nil {
						t.Fatal(err)
					}
					records = append(records, v)
				}
				stage, ok := records[4]["stage"].(map[string]any)
				if records[4]["msg"] != "got rate limited, sleeping" || records[4]["job"] != "example" || !ok || stage["channel"] != "C123" || stage["source"] != "Slack API" {
					t.Fatalf("structured record changed: %v", records[4])
				}
				warning, ok := records[5]["stage"].(map[string]any)
				if !ok || warning["error"] != "private wrapped cause" {
					t.Fatalf("full error missing from file: %v", records[5])
				}
			}
		})
	}
	t.Run("native console style without added timestamp or attributes", func(t *testing.T) {
		r := testReporter()
		var actual, expected bytes.Buffer
		r.output = &actual
		h := &handler{next: slog.NewTextHandler(&actual, nil), console: cfg.ConsoleLogHandler(reporterOutput{r}), r: r}
		rec := slog.NewRecord(time.Now(), slog.LevelWarn, "canvas discussions unavailable", 0)
		rec.AddAttrs(slog.String("channel_id", "C123"), slog.String("error", "wrapped cause"))
		if err := h.Handle(t.Context(), rec); err != nil {
			t.Fatal(err)
		}
		plain := slog.NewRecord(rec.Time, rec.Level, rec.Message, rec.PC)
		if err := cfg.ConsoleLogHandler(&expected).Handle(t.Context(), plain); err != nil {
			t.Fatal(err)
		}
		normalize := func(s string) string {
			s = ansi.Strip(s)
			return regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`).ReplaceAllString(s, "<timestamp>")
		}
		got, want := normalize(actual.String()), normalize(expected.String())
		if got != want || strings.Count(got, "\n") != 1 || strings.Contains(got, "C123") || strings.Contains(got, "wrapped cause") {
			t.Fatalf("console=%q native=%q", got, want)
		}
	})
	t.Run("progress-only warning description preserves file record", func(t *testing.T) {
		r := testReporter()
		var terminal, file bytes.Buffer
		r.output = &terminal
		r.separateLog = true
		logger := slog.New(&handler{next: slog.NewJSONHandler(&file, nil), console: slog.NewTextHandler(&terminal, nil), r: r})
		ctx := progresslog.WithMessage(t.Context(), "skipping non-existing thread in #creditcards\x1b[31m\nforged")
		logger.WarnContext(ctx, "skipping non-existing thread", "channel_id", "C123", "thread_ts", "1.0")
		text := terminal.String()
		if strings.Count(text, "\n") != 1 || strings.Contains(text, "\x1b") || !strings.Contains(text, "skipping non-existing thread in #creditcards forged") || strings.Contains(text, "C123") || strings.Contains(text, "1.0") {
			t.Fatalf("bad terminal warning: %q", text)
		}
		var record map[string]any
		if err := json.Unmarshal(file.Bytes(), &record); err != nil {
			t.Fatal(err)
		}
		if record["msg"] != "skipping non-existing thread" || record["channel_id"] != "C123" || record["thread_ts"] != "1.0" {
			t.Fatalf("file warning changed: %v", record)
		}
	})
	t.Run("control sequences cannot forge console lines", func(t *testing.T) {
		r := testReporter()
		var terminal, file bytes.Buffer
		r.output = &terminal
		r.separateLog = true
		h := &handler{next: slog.NewTextHandler(&file, nil), console: slog.NewTextHandler(&terminal, nil), r: r}
		rec := slog.NewRecord(time.Date(2026, 11, 14, 8, 9, 10, 0, time.UTC), slog.LevelWarn, "canvas discussions unavailable\x1b[31m\nforged\u2028line", 0)
		rec.AddAttrs(slog.String("error", "wrapped cause"))
		if err := h.Handle(t.Context(), rec); err != nil {
			t.Fatal(err)
		}
		got := terminal.String()
		if strings.Count(got, "\n") != 1 || strings.Contains(got, "\x1b") || strings.Contains(got, "wrapped cause") || strings.Contains(got, "[2026-11-14") || !strings.Contains(got, "canvas discussions unavailable forged line") {
			t.Fatalf("unsafe console output %q", got)
		}
		if !strings.Contains(file.String(), "wrapped cause") {
			t.Fatalf("file lost error detail: %q", file.String())
		}
	})
	t.Run("event log lines are not limited by panel width", func(t *testing.T) {
		r := testReporter()
		r.tty = true
		r.width = func() int { return 80 }
		var out bytes.Buffer
		r.output = &out
		h := &handler{next: slog.NewTextHandler(&out, nil), console: slog.NewTextHandler(&out, nil), r: r}
		message := strings.Repeat("x", 120)
		rec := slog.NewRecord(time.Now(), slog.LevelWarn, message, 0)
		if err := h.Handle(t.Context(), rec); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), message) {
			t.Fatalf("log event was truncated to the panel width: %q", out.String())
		}
	})
	t.Run("logs redraw the TTY panel", func(t *testing.T) {
		r := testReporter()
		var out bytes.Buffer
		r.output = &out
		r.tty = true
		h := &handler{next: slog.NewTextHandler(&out, nil), console: slog.NewTextHandler(&out, nil), r: r}
		r.draw(false)
		rec := slog.NewRecord(time.Date(2026, 11, 14, 8, 9, 10, 0, time.UTC), slog.LevelWarn, "warning", 0)
		if err := h.Handle(t.Context(), rec); err != nil {
			t.Fatal(err)
		}
		text := out.String()
		if strings.Contains(text, "[2026-11-14 08:09]") || !strings.Contains(text, "level=WARN msg=warning") || !strings.Contains(text, "\x1b[3A\r\x1b[J") || strings.Count(text, "Runtime") != 2 {
			t.Fatalf("log/panel output %q", text)
		}
	})
}

func Test_bar(t *testing.T) {
	for _, tt := range []struct {
		done, total, width int
		want               string
	}{
		{0, 0, 0, "[]"}, {0, 0, 8, "[--------]"}, {3, 21, 14, "[##------------]"}, {6, 10, 10, "[######----]"}, {1, 3, -1, "[]"},
	} {
		if got := bar(tt.done, tt.total, tt.width); got != tt.want {
			t.Errorf("bar(%d,%d,%d)=%q want %q", tt.done, tt.total, tt.width, got, tt.want)
		}
	}
}

func TestReporter_lines(t *testing.T) {
	r := testReporter()
	r.tty = true
	r.channels["C123"] = &job{pending: 1}
	r.channels["other"] = &job{}
	r.threads[key{"C123", "1"}] = &job{}
	r.threads[key{"C123", "2"}] = &job{pending: 1}
	lines := r.lines(80)
	if !strings.Contains(lines[0], "1/2 (discovering)") || !strings.Contains(lines[1], "1/2 (discovering)") || strings.Contains(strings.Join(lines, ""), "C123") || strings.Contains(strings.Join(lines, ""), "Starting") {
		t.Fatalf("unexpected panel: %v", lines)
	}
	r.threads[key{"C123", "3"}] = &job{pending: 1}
	lines = r.lines(80)
	if !strings.Contains(lines[1], "1/3 (discovering)") || strings.Contains(lines[1], "complete") {
		t.Fatalf("growing denominator or labels: %v", lines)
	}
	r.Event(stream.ProgressEvent{Kind: stream.ProgressInputDone})
	r.Event(stream.ProgressEvent{Kind: stream.ProgressDiscoveryDone})
	lines = r.lines(80)
	if strings.Contains(lines[0], "discovering") || strings.Contains(lines[1], "discovering") || !strings.Contains(lines[1], "1/3") {
		t.Fatalf("frozen counts: %v", lines)
	}
	r.waits[1] = time.Now().Add(time.Minute)
	lines = r.lines(80)
	if !strings.Contains(lines[2], "rate-limited, retry in") {
		t.Fatalf("rate-wait lost: %v", lines)
	}
}

func TestReporter_drawResize(t *testing.T) {
	if os.Getenv("SLACKDUMP_RESIZE_CHILD") == "1" {
		resizeTerminalChild(t)
		return
	}
	t.Run("terminal reflow preserves prior logs", resizeTerminalTest)
	r := testReporter()
	r.tty = true
	r.channels["C"] = &job{pending: 1}
	r.threads[key{"C", "1"}] = &job{}
	width := 60
	r.width = func() int { return width }
	var out bytes.Buffer
	r.output = &out
	r.draw(true)
	small := strings.Split(out.String(), "\n")[0]
	out.Reset()
	width = 120
	r.draw(true)
	wide := strings.Split(out.String(), "\n")[0]
	size := func(s string) int {
		start, end := strings.Index(s, "["), strings.Index(s, "]")
		if start < 0 || end < start {
			t.Fatalf("missing bar: %q", s)
		}
		return end - start - 1
	}
	if size(small) < 1 || size(wide) <= size(small) {
		t.Fatalf("bars did not widen: small=%q wide=%q", small, wide)
	}
	for _, w := range []int{1, 8, 20, 30, 40, 60, 80, 120} {
		width = w
		out.Reset()
		r.draw(true)
		lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("width %d: %q", w, out.String())
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > panelWidth(w, true) {
				t.Fatalf("width %d overflow: %q", w, line)
			}
		}
	}
}

func Test_panelWidth(t *testing.T) {
	for _, tt := range []struct {
		width int
		tty   bool
		want  int
	}{
		{80, true, 79}, {100, true, 80}, {30, true, 29}, {1, true, 0}, {0, true, 0}, {-5, true, 0}, {1, false, 80}, {120, false, 80},
	} {
		if got := panelWidth(tt.width, tt.tty); got != tt.want {
			t.Errorf("panelWidth(%d,%t)=%d want %d", tt.width, tt.tty, got, tt.want)
		}
	}
}

func TestFlags_Begin(t *testing.T) {
	var f Flags
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	f.Bind(fs)
	if err := fs.Parse([]string{"--progress-interval=0"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.Begin(t.Context()); err == nil {
		t.Fatal("zero interval accepted")
	}
	f.Interval = -time.Second
	if _, _, err := f.Begin(t.Context()); err == nil {
		t.Fatal("negative interval accepted")
	}
	f.Interval = time.Second
	f.Enabled = false
	beforeCfg, beforeDefault := cfg.Log, slog.Default()
	if r, _, err := f.Begin(t.Context()); err != nil || r != nil {
		t.Fatalf("disabled: %v %v", r, err)
	}
	if cfg.Log != beforeCfg || slog.Default() != beforeDefault {
		t.Fatal("disabled progress changed logging")
	}
	oldJSON, oldFile, oldStderr := cfg.UseJSONHandler, cfg.LogFile, os.Stderr
	defer func() { cfg.UseJSONHandler, cfg.LogFile = oldJSON, oldFile }()
	cfg.UseJSONHandler = true
	cfg.LogFile = ""
	f.Enabled = true
	if _, _, err := f.Begin(context.Background()); err == nil {
		t.Fatal("shared JSON stderr accepted")
	}
	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer stderr.Close()
	os.Stderr = stderr
	defer func() { os.Stderr = oldStderr }()
	cfg.LogFile = stderr.Name()
	if r, _, err := f.Begin(context.Background()); err == nil {
		if r != nil {
			r.Stop()
		}
		t.Fatal("aliased JSON stderr accepted")
	}
}

func TestStart(t *testing.T) {
	if mode := os.Getenv("SLACKDUMP_PROGRESS_CHILD"); mode == "3" || mode == "4" {
		slog.Warn("native sample")
		var file bytes.Buffer
		if mode == "4" {
			logger := slog.New(slog.NewJSONHandler(&file, nil))
			cfg.Log = logger
			slog.SetDefault(logger)
			cfg.UseJSONHandler = true
			cfg.LogFile = t.TempDir() + "/separate.json"
		}
		r, _, err := Start(t.Context(), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		slog.Warn("native sample", "error", "private cause")
		r.Stop()
		if mode == "4" && !strings.Contains(file.String(), "private cause") {
			t.Fatal("file lost warning attributes")
		}
		return
	}
	if os.Getenv("SLACKDUMP_PROGRESS_CHILD") == "2" {
		var file bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&file, nil))
		cfg.Log = logger
		slog.SetDefault(logger)
		cfg.UseJSONHandler = true
		cfg.LogFile = t.TempDir() + "/separate.json"
		r, ctx, err := Start(t.Context(), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		done := network.ObserveRateLimit(ctx, "Slack API", 7*time.Second)
		done()
		slog.Info("stream")
		r.Stop()
		if strings.Count(file.String(), "rate limited") != 1 || !strings.Contains(file.String(), "stream") {
			t.Fatal("structured file output incomplete")
		}
		return
	}
	if os.Getenv("SLACKDUMP_PROGRESS_CHILD") == "1" {
		r, _, err := Start(t.Context(), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		slog.Warn("built-in warning")
		log.Print("standard log line")
		r.Stop()
		return
	}
	for _, mode := range []string{"3", "4"} {
		t.Run("no color matches original builtin console "+mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStart$")
			command.Env = append(os.Environ(), "NOCOLOR=true", "SLACKDUMP_PROGRESS_CHILD="+mode)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("child: %v: %s", err, output)
			}
			var lines []string
			for _, line := range strings.Split(string(output), "\n") {
				if strings.Contains(line, "native sample") {
					lines = append(lines, line)
				}
			}
			if len(lines) != 2 {
				t.Fatalf("expected two native warnings: %q", output)
			}
			normalize := func(line string) string {
				return regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`).ReplaceAllString(line, "<timestamp>")
			}
			if normalize(lines[0]) != normalize(lines[1]) || !strings.Contains(lines[1], " WARN native sample") || strings.Contains(lines[1], "private cause") {
				t.Fatalf("builtin=%q progress=%q", lines[0], lines[1])
			}
		})
	}
	t.Run("rate notices distinguish waiting from direct failures", func(t *testing.T) {
		oldCfg, oldDefault, oldJSON, oldFile := cfg.Log, slog.Default(), cfg.UseJSONHandler, cfg.LogFile
		defer func() {
			cfg.Log = oldCfg
			slog.SetDefault(oldDefault)
			cfg.UseJSONHandler = oldJSON
			cfg.LogFile = oldFile
		}()
		var file, terminal bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&file, nil))
		cfg.Log = logger
		slog.SetDefault(logger)
		cfg.UseJSONHandler = true
		cfg.LogFile = t.TempDir() + "/separate.json"
		r, ctx, err := Start(t.Context(), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		r.output = &terminal
		r.tty = false
		done := network.ObserveRateLimitBeforeWait(ctx, "Slack API", 7*time.Second)
		done()
		done = network.ObserveRateLimit(ctx, "Slack API", 7*time.Second)
		done()
		r.Stop()
		text := terminal.String()
		if strings.Count(text, "got rate limited, sleeping") != 1 || strings.Count(text, "got rate limited") != 2 || strings.Contains(text, "Slack API") || strings.Contains(text, "7s") {
			t.Fatalf("unexpected rate output %q", text)
		}
		if strings.Count(file.String(), "retry_after") != 2 || strings.Count(file.String(), "Slack API") != 2 {
			t.Fatalf("rate details missing from file %q", file.String())
		}
	})
	t.Run("no color separate log mirrors warning", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStart$")
		command.Env = append(os.Environ(), "NOCOLOR=true", "SLACKDUMP_PROGRESS_CHILD=2")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("child: %v: %s", err, output)
		}
		text := string(output)
		if strings.Count(text, "rate limited") != 1 || strings.Contains(text, "Slack API") || strings.Contains(text, "7s") || strings.Contains(text, "stream") {
			t.Fatalf("unexpected console output %q", text)
		}
	})
	t.Run("separate JSON log mirrors cfg and default warnings once", func(t *testing.T) {
		oldCfg, oldDefault, oldJSON, oldFile, oldVerbose := cfg.Log, slog.Default(), cfg.UseJSONHandler, cfg.LogFile, cfg.Verbose
		defer func() {
			cfg.Log = oldCfg
			slog.SetDefault(oldDefault)
			cfg.UseJSONHandler = oldJSON
			cfg.LogFile = oldFile
			cfg.Verbose = oldVerbose
		}()
		var file, terminal bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&file, &slog.HandlerOptions{Level: slog.LevelDebug}))
		cfg.Log = logger
		slog.SetDefault(logger)
		cfg.UseJSONHandler = true
		cfg.LogFile = t.TempDir() + "/separate.json"
		cfg.Verbose = true
		r, ctx, err := Start(t.Context(), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		r.output = &terminal
		r.tty = false
		cfg.Log.With("origin", "cfg").WithGroup("job").Warn("cfg warning", "channel", "C1")
		slog.With("origin", "default").WithGroup("job").Error("default error", "channel", "C2")
		slog.Info("stream")
		done := network.ObserveRateLimit(ctx, "Slack API", 7*time.Second)
		done()
		r.Stop()
		text := terminal.String()
		for _, want := range []string{"cfg warning", "default error", "rate limited"} {
			if !strings.Contains(text, want) {
				t.Fatalf("missing %q in %q", want, text)
			}
		}
		if strings.Count(text, "rate limited") != 1 || strings.Contains(text, "stream") || strings.Contains(text, "Slack API") || strings.Contains(text, "7s") || strings.Contains(text, "C1") || strings.Contains(text, "C2") {
			t.Fatalf("unexpected terminal output: %q", text)
		}
		rows := strings.Split(strings.TrimSpace(file.String()), "\n")
		if len(rows) != 4 {
			t.Fatalf("unexpected log records %q", file.String())
		}
		for _, row := range rows {
			var v map[string]any
			if err := json.Unmarshal([]byte(row), &v); err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Run("text stderr alias is not mirrored", func(t *testing.T) {
		oldCfg, oldDefault, oldJSON, oldFile, oldStderr := cfg.Log, slog.Default(), cfg.UseJSONHandler, cfg.LogFile, os.Stderr
		f, err := os.CreateTemp(t.TempDir(), "stderr")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		os.Stderr = f
		defer func() {
			os.Stderr = oldStderr
			cfg.Log = oldCfg
			slog.SetDefault(oldDefault)
			cfg.UseJSONHandler = oldJSON
			cfg.LogFile = oldFile
		}()
		cfg.UseJSONHandler = false
		cfg.LogFile = f.Name()
		logger := slog.New(slog.NewTextHandler(f, nil))
		cfg.Log = logger
		slog.SetDefault(logger)
		r, _, err := Start(t.Context(), time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if r.separateLog {
			t.Fatal("stderr alias treated as separate")
		}
		r.tty = false
		slog.Warn("alias warning")
		r.Stop()
		data, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if strings.Count(text, "alias warning") != 1 || strings.Contains(text, "[2026-") {
			t.Fatalf("alias output %q", text)
		}
	})
	t.Run("no color built-in handler does not recurse", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStart$")
		command.Env = append(os.Environ(), "NOCOLOR=true", "SLACKDUMP_PROGRESS_CHILD=1")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("child: %v: %s", err, output)
		}
		if !bytes.Contains(output, []byte("built-in warning")) || bytes.Contains(output, []byte("standard log line")) {
			t.Fatalf("missing log output: %s", output)
		}
	})
	oldCfg, oldDefault, oldJSON, oldFile := cfg.Log, slog.Default(), cfg.UseJSONHandler, cfg.LogFile
	defer func() {
		cfg.Log = oldCfg
		slog.SetDefault(oldDefault)
		cfg.UseJSONHandler = oldJSON
		cfg.LogFile = oldFile
	}()
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	cfg.Log = logger
	slog.SetDefault(logger)
	cfg.UseJSONHandler = true
	cfg.LogFile = t.TempDir() + "/separate.log"
	r, ctx, err := Start(t.Context(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	stopped := false
	defer func() {
		if !stopped {
			r.Stop()
		}
	}()
	var snapshots bytes.Buffer
	r.output = &snapshots
	r.tty = false
	done := network.ObserveRateLimit(ctx, "test API", 5*time.Second)
	done()
	notice := snapshots.String()
	if strings.Count(notice, "got rate limited") != 1 || strings.Contains(notice, "test API") || strings.Contains(notice, "5s") {
		t.Fatalf("unexpected mirrored rate notice: %q", notice)
	}
	if strings.Count(ansi.Strip(notice), "got rate limited") != 1 {
		t.Fatalf("rate notice was not mirrored once: %q", notice)
	}
	snapshots.Reset()
	r.Stop()
	stopped = true
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &record); err != nil {
		t.Fatalf("invalid JSON: %v: %q", err, logs.String())
	}
	if record["level"] != "INFO" || record["source"] != "test API" {
		t.Fatalf("unexpected warning: %v", record)
	}
	if strings.Contains(snapshots.String(), "\x1b") || !strings.Contains(snapshots.String(), "Runtime") || strings.Contains(snapshots.String(), "cancelled") {
		t.Fatalf("bad final snapshot: %q", snapshots.String())
	}
	select {
	case <-r.done:
	default:
		t.Fatal("renderer was not joined")
	}
}
