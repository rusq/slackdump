// Package progress reports optional, run-scoped streaming progress.
package progress

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/progresslog"
	"github.com/rusq/slackdump/v4/stream"
)

// Flags configures progress reporting for a command.
type Flags struct {
	Enabled  bool
	Interval time.Duration
}

func (f *Flags) Bind(fs *flag.FlagSet) {
	fs.BoolVar(&f.Enabled, "progress", false, "show archive progress on stderr")
	fs.DurationVar(&f.Interval, "progress-interval", 5*time.Second, "progress refresh interval (positive duration)")
}

type contextKey struct{}
type rateLimitLogKey struct{}

func rateLimitLogContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, rateLimitLogKey{}, true)
}

// Begin validates the flags and starts a reporter when enabled.
func (f Flags) Begin(ctx context.Context) (*Reporter, context.Context, error) {
	if f.Interval <= 0 {
		return nil, ctx, fmt.Errorf("progress interval must be positive")
	}
	if !f.Enabled {
		return nil, ctx, nil
	}
	// The CLI may already own a run started before authentication. Reuse it;
	// only its original owner stops the reporter.
	if _, ok := ctx.Value(contextKey{}).(*Reporter); ok {
		return nil, ctx, nil
	}
	r, ctx, err := Start(ctx, f.Interval)
	if err != nil {
		return nil, ctx, err
	}
	return r, context.WithValue(ctx, contextKey{}, r), nil
}

// Options returns streaming hooks for the current run, if enabled.
func Options(ctx context.Context) []stream.Option {
	if r, ok := ctx.Value(contextKey{}).(*Reporter); ok {
		return []stream.Option{stream.OptProgress(r.Event)}
	}
	return nil
}

type job struct {
	pending, callbacks int
	failed             bool
}
type key struct{ channel, thread string }

// Reporter owns the state and renderer for one run.
type Reporter struct {
	mu                 sync.Mutex
	channels           map[string]*job
	threads            map[key]*job
	waits              map[uint64]time.Time
	nextWait           uint64
	discovering        bool
	channelDiscovering bool
	start              time.Time
	output             io.Writer
	tty                bool
	width              func() int
	drawMu             sync.Mutex
	drawn              bool
	panelLines         []string
	stop               chan struct{}
	done               chan struct{}
	oldDefault, oldCfg *slog.Logger
	oldLogWriter       io.Writer
	oldLogFlags        int
	separateLog        bool
}

// Start initializes progress output and wraps the existing logger without changing its destination.
func Start(ctx context.Context, interval time.Duration) (*Reporter, context.Context, error) {
	if interval <= 0 {
		return nil, ctx, fmt.Errorf("progress interval must be positive")
	}
	sameStream := cfg.LogFile == ""
	if !sameStream {
		if fileInfo, err := os.Stat(cfg.LogFile); err == nil {
			if stderrInfo, err := os.Stderr.Stat(); err == nil {
				sameStream = os.SameFile(fileInfo, stderrInfo)
			}
		}
	}
	if cfg.UseJSONHandler && sameStream {
		return nil, ctx, fmt.Errorf("--progress cannot share stderr with --log-json; use --log to write JSON to a file")
	}
	r := &Reporter{channels: make(map[string]*job), threads: make(map[key]*job), waits: make(map[uint64]time.Time), discovering: true, channelDiscovering: true, start: time.Now(), output: os.Stderr, tty: term.IsTerminal(int(os.Stderr.Fd())), stop: make(chan struct{}), done: make(chan struct{}), oldDefault: slog.Default(), oldCfg: cfg.Log, oldLogWriter: log.Writer(), oldLogFlags: log.Flags(), separateLog: !sameStream}
	r.width = func() int {
		w, _, err := term.GetSize(int(os.Stderr.Fd()))
		if err != nil || w < 1 {
			return 80
		}
		return w
	}
	// Existing cfg.Log and default logger normally share a handler, but wrap each
	// independently so their configured routing and attributes remain intact.
	wrap := func(h slog.Handler) slog.Handler {
		console := cfg.ConsoleLogHandler(reporterOutput{r})
		return &handler{next: h, console: console, r: r}
	}
	cfg.Log = slog.New(wrap(r.oldCfg.Handler()))
	slog.SetDefault(slog.New(wrap(r.oldDefault.Handler())))
	ctx = network.WithRateLimitObserver(ctx, func(source string, delay time.Duration, warning, willWait bool) func() {
		message := "got rate limited"
		if willWait {
			message = "got rate limited, sleeping"
		}
		if delay < 0 {
			if warning {
				slog.InfoContext(rateLimitLogContext(ctx), message, "source", source, "retry_after", "unknown")
			}
			return func() {}
		}
		r.mu.Lock()
		r.nextWait++
		id := r.nextWait
		r.waits[id] = time.Now().Add(delay)
		r.mu.Unlock()
		if warning {
			slog.InfoContext(rateLimitLogContext(ctx), message, "source", source, "retry_after", delay)
		}
		return func() { r.mu.Lock(); delete(r.waits, id); r.mu.Unlock() }
	})
	go func() {
		defer close(r.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.draw(false)
			case <-r.stop:
				return
			}
		}
	}()
	return r, ctx, nil
}

// Event records the actual pipeline lifecycle without affecting work scheduling.
func (r *Reporter) Event(e stream.ProgressEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch e.Kind {
	case stream.ProgressInputDone:
		if e.Err == nil {
			r.channelDiscovering = false
		}
		return
	case stream.ProgressDiscoveryDone:
		if e.Err == nil {
			r.discovering = false
		}
		return
	}

	var j *job
	switch e.Kind {
	case stream.ProgressChannelQueued, stream.ProgressChannelDone:
		j = r.channels[e.ChannelID]
		if j == nil && e.Kind == stream.ProgressChannelQueued {
			j = new(job)
			r.channels[e.ChannelID] = j
		}
	case stream.ProgressThreadQueued, stream.ProgressThreadDone:
		k := key{e.ChannelID, e.ThreadTS}
		j = r.threads[k]
		if j == nil && e.Kind == stream.ProgressThreadQueued {
			j = new(job)
			r.threads[k] = j
		}
	case stream.ProgressResultQueued, stream.ProgressResultProcessed:
		if e.ThreadTS == "" {
			j = r.channels[e.ChannelID]
		} else {
			j = r.threads[key{e.ChannelID, e.ThreadTS}]
		}
	}
	if j == nil {
		return
	}
	switch e.Kind {
	case stream.ProgressChannelQueued, stream.ProgressThreadQueued:
		j.pending++
	case stream.ProgressChannelDone, stream.ProgressThreadDone:
		j.pending--
		j.failed = j.failed || e.Err != nil
	case stream.ProgressResultQueued:
		j.callbacks++
	case stream.ProgressResultProcessed:
		if j.callbacks > 0 {
			j.callbacks--
		}
		j.failed = j.failed || e.Err != nil
	}
}

// Stop joins the renderer and prints one final snapshot after all work has drained.
func (r *Reporter) Stop() {
	close(r.stop)
	<-r.done
	r.draw(true)
	cfg.Log = r.oldCfg
	slog.SetDefault(r.oldDefault)
	log.SetOutput(r.oldLogWriter)
	log.SetFlags(r.oldLogFlags)
}
func count[K comparable](jobs map[K]*job) (done, failed int) {
	for _, j := range jobs {
		if j.failed {
			failed++
		} else if j.pending == 0 && j.callbacks == 0 {
			done++
		}
	}
	return
}
func bar(done, total, width int) string {
	width = max(0, width)
	filled := 0
	if total > 0 {
		filled = width * done / total
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func countLabel(done, total, failed int, discovering bool) string {
	label := fmt.Sprintf("%d/%d", done, total)
	if discovering {
		label += " (discovering)"
	}
	if failed > 0 {
		label += fmt.Sprintf("; %d failed", failed)
	}
	return label
}

func elapsed(d time.Duration) string {
	seconds := int64(d.Seconds())
	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, (seconds/60)%60, seconds%60)
}

// lines returns the compact panel for the given column budget.
func (r *Reporter) lines(width int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	cd, cf := count(r.channels)
	td, tf := count(r.threads)
	chCount := countLabel(cd, len(r.channels), cf, r.channelDiscovering)
	thCount := countLabel(td, len(r.threads), tf, r.discovering)
	const fixedWidth = len("Channels  ") + 2 + 1
	barWidth := max(0, width-fixedWidth-max(ansi.StringWidth(chCount), ansi.StringWidth(thCount)))
	ch := "Channels  " + bar(cd, len(r.channels), barWidth) + " " + chCount
	th := "Threads   " + bar(td, len(r.threads), barWidth) + " " + thCount
	runtime := "Runtime   " + elapsed(time.Since(r.start))
	if len(r.waits) > 0 {
		var earliest time.Time
		for _, until := range r.waits {
			if earliest.IsZero() || until.Before(earliest) {
				earliest = until
			}
		}
		left := time.Until(earliest).Round(time.Second)
		if left < 0 {
			left = 0
		}
		runtime += fmt.Sprintf(" | %d rate-limited, retry in %s", len(r.waits), left)
	}
	return []string{ch, th, runtime}
}

// clear removes the previous panel, accounting for terminal reflow after a resize.
// width is the raw current terminal width, not the configured panel budget.
func (r *Reporter) clear(width int) {
	if r.tty && r.drawn {
		rows := 0
		for _, line := range r.panelLines {
			rows += 1 + strings.Count(ansi.Hardwrap(line, max(1, width), true), "\n")
		}
		fmt.Fprintf(r.output, "\x1b[%dA\r\x1b[J", rows)
		r.drawn = false
		r.panelLines = nil
	}
}
func (r *Reporter) draw(final bool) { r.drawMu.Lock(); defer r.drawMu.Unlock(); r.drawLocked(final) }

// panelWidth caps a terminal panel before its right edge; redirected snapshots
// use the configured width without assuming a physical screen.
func panelWidth(terminalWidth int, tty bool) int {
	if tty {
		return min(80, max(0, terminalWidth-1))
	}
	return 80
}

func (r *Reporter) drawLocked(final bool) {
	rawWidth := r.width()
	r.clear(rawWidth)
	width := panelWidth(rawWidth, r.tty)
	lines := r.lines(width)
	rendered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = ansi.Truncate(progresslog.Sanitize(line), width, "")
		fmt.Fprintln(r.output, line)
		rendered = append(rendered, line)
	}
	r.drawn = r.tty && !final
	if r.drawn {
		r.panelLines = rendered
	}
}

// reporterOutput follows the reporter's output writer, which also makes
// mirrored logging use the same destination as the progress panel.
type reporterOutput struct{ r *Reporter }

func (w reporterOutput) Write(p []byte) (int, error) { return w.r.output.Write(p) }

type handler struct {
	next    slog.Handler
	console slog.Handler
	r       *Reporter
}

func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level) || h.console.Enabled(ctx, level)
}

// terminalVisible reports warnings, errors, and the typed rate-limit event.
func terminalVisible(ctx context.Context, rec slog.Record) bool {
	if rec.Level >= slog.LevelWarn {
		return true
	}
	marked, _ := ctx.Value(rateLimitLogKey{}).(bool)
	return rec.Level == slog.LevelInfo && marked
}

func (h *handler) Handle(ctx context.Context, rec slog.Record) error {
	visible := terminalVisible(ctx, rec)
	// A separate logfile retains the original record and its configured format.
	var err error
	if h.r.separateLog && h.next.Enabled(ctx, rec.Level) {
		err = h.next.Handle(ctx, rec)
	}
	if !visible {
		return err
	}
	h.r.drawMu.Lock()
	defer h.r.drawMu.Unlock()
	had := h.r.drawn
	h.r.clear(h.r.width())
	message := rec.Message
	if concise, ok := progresslog.Message(ctx); ok {
		message = concise
	}
	message = progresslog.Sanitize(message)
	// A fresh record prevents attributes or bound groups from leaking to the
	// console while keeping the console logger's native timestamp and level.
	consoleRecord := slog.NewRecord(rec.Time, rec.Level, message, rec.PC)
	if consoleErr := h.console.Handle(ctx, consoleRecord); err == nil {
		err = consoleErr
	}
	if had {
		h.r.drawLocked(false)
	}
	return err
}
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{next: h.next.WithAttrs(attrs), console: h.console, r: h.r}
}
func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{next: h.next.WithGroup(name), console: h.console, r: h.r}
}
