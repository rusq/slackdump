//go:build ignore

package ui

import (
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
	"github.com/kr/pty"
)

type console interface {
	ExpectString(s string) (string, error)
	SendLine(s string) (int, error)
	ExpectEOF() (string, error)
}

// RunTest is the helper function to execute the UI tests.  procedure is the
// function that contains Expect interactions with the UI, and test is the
// function that should invoke the UI element.
//
// It's a simplified copy/paste from the survey lib:
//
//	https://github.com/go-survey/survey/blob/master/survey_posix_test.go
func RunTest(t *testing.T, procedure func(*testing.T, console), test func(terminal.Stdio) error) {
	t.Helper()

	pty, tty, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	term := vt10x.New(vt10x.WithWriter(tty))
	console, err := expect.NewConsole(expect.WithStdin(pty), expect.WithStdout(term), expect.WithCloser(pty, tty))
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		procedure(t, console)
	}()
	stdio := terminal.Stdio{In: console.Tty(), Out: console.Tty(), Err: console.Tty()}
	if err := test(stdio); err != nil {
		t.Error(err)
	}
	if err := console.Tty().Close(); err != nil {
		t.Errorf("error closing tty: %s", err)
	}
	<-done
}
