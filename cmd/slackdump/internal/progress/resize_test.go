package progress

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/term"
)

// resizeTerminalChild runs inside an isolated tmux pane. Its output exercises
// the same draw and log-redraw paths as the command, without a live service.
func resizeTerminalChild(t *testing.T) {
	stty := exec.Command("stty", "-echo")
	stty.Stdin = os.Stdin
	if err := stty.Run(); err != nil {
		t.Fatal(err)
	}
	signal := func(name string) {
		t.Helper()
		if out, err := exec.Command("tmux", "-S", os.Getenv("SLACKDUMP_RESIZE_SOCKET"), "wait-for", "-S", name).CombinedOutput(); err != nil {
			t.Fatalf("signal %s: %v %s", name, err, out)
		}
	}
	r := testReporter()
	r.tty = true
	r.output = os.Stdout
	r.width = func() int {
		w, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return 80
		}
		return w
	}
	r.channels["C"] = &job{pending: 1}
	r.threads[key{"C", "1"}] = &job{}
	console := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey || a.Key == slog.LevelKey {
			return slog.Attr{}
		}
		return a
	}})
	h := &handler{next: slog.NewTextHandler(io.Discard, nil), console: console, r: r}
	fmt.Fprintln(os.Stdout, "KEEP-PRIOR-LOG")
	r.draw(false)
	signal("ready")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "resize ") {
			want, _ := strconv.Atoi(strings.TrimPrefix(scanner.Text(), "resize "))
			deadline := time.Now().Add(3 * time.Second)
			for r.width() != want && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			if r.width() != want {
				t.Fatalf("terminal width = %d, want %d", r.width(), want)
			}
			signal("done")
			continue
		}
		switch scanner.Text() {
		case "warning1", "warning2":
			rec := slog.NewRecord(time.Now(), slog.LevelWarn, "KEEP-"+strings.ToUpper(scanner.Text()), 0)
			if err := h.Handle(context.Background(), rec); err != nil {
				t.Fatal(err)
			}
		case "draw":
			r.draw(false)
		case "quit":
			return
		}
		signal("done")
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

// resizeTerminalTest uses tmux's actual reflow behavior on a private socket;
// it never attaches to or touches a user's existing tmux server.
func resizeTerminalTest(t *testing.T) {
	tmux, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux is not installed")
	}
	socket := filepath.Join(t.TempDir(), "tmux.sock")
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.CommandContext(ctx, tmux, append([]string{"-S", socket}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("tmux %v: %v: %s", args, err, out)
		}
		return string(out)
	}
	create := exec.CommandContext(ctx, tmux, "-f", "/dev/null", "-S", socket, "new-session", "-d", "-x", "100", "-y", "25", "-s", "progress-resize", "env", "SLACKDUMP_RESIZE_CHILD=1", "SLACKDUMP_RESIZE_SOCKET="+socket, os.Args[0], "-test.run=^TestReporter_drawResize$")
	if out, err := create.CombinedOutput(); err != nil {
		t.Fatalf("create private tmux: %v: %s", err, out)
	}
	defer exec.Command(tmux, "-S", socket, "kill-server").Run()
	run("wait-for", "ready")
	run("set-option", "-g", "status", "off")
	run("set-window-option", "-t", "progress-resize:0", "window-size", "manual")
	check := func(stage string, warnings int) {
		t.Helper()
		screen := run("capture-pane", "-p", "-t", "progress-resize:0.0")
		history := run("capture-pane", "-p", "-t", "progress-resize:0.0", "-S", "-100")
		if strings.Count(history, "KEEP-PRIOR-LOG") != 1 || strings.Count(screen, "Channels") != 1 || strings.Count(screen, "Threads") != 1 || strings.Count(screen, "Runtime") != 1 || strings.Count(history, "KEEP-WARNING") != warnings {
			t.Fatalf("%s: stale panel or lost log:\n%s\nhistory:\n%s", stage, screen, history)
		}
	}
	run("resize-window", "-t", "progress-resize:0", "-x", "85", "-y", "25")
	run("send-keys", "-t", "progress-resize:0.0", "resize 85", "Enter")
	run("wait-for", "done")
	run("send-keys", "-t", "progress-resize:0.0", "warning1", "Enter")
	run("wait-for", "done")
	check("shrink without wrapping", 1)
	run("resize-window", "-t", "progress-resize:0", "-x", "30", "-y", "25")
	run("send-keys", "-t", "progress-resize:0.0", "resize 30", "Enter")
	run("wait-for", "done")
	run("send-keys", "-t", "progress-resize:0.0", "warning2", "Enter")
	run("wait-for", "done")
	check("shrink with reflow", 2)
	run("resize-window", "-t", "progress-resize:0", "-x", "100", "-y", "25")
	run("send-keys", "-t", "progress-resize:0.0", "resize 100", "Enter")
	run("wait-for", "done")
	run("send-keys", "-t", "progress-resize:0.0", "draw", "Enter")
	run("wait-for", "done")
	check("grow after reflow", 2)
	run("send-keys", "-t", "progress-resize:0.0", "quit", "Enter")
}
