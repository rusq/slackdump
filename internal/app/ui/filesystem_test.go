package ui

import (
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/stretchr/testify/assert"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestFileselector(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		var filename string
		testFn := func(stdio terminal.Stdio) error {
			var err error
			filename, err = FileSelector("xxx", "help", WithOutput(stdio))
			return err
		}
		procedure := func(t *testing.T, console *expect.Console) {
			console.ExpectString("xxx")
			console.SendLine("test.txt")
			console.ExpectEOF()
		}
		RunTest(t, procedure, testFn)
		assert.Equal(t, "test.txt", filename)
	})
	t.Run("empty with no override", func(t *testing.T) {
		var filename string
		testFn := func(stdio terminal.Stdio) error {
			var err error
			filename, err = FileSelector("xxx", "help", WithOutput(stdio))
			return err
		}
		procedure := func(t *testing.T, console *expect.Console) {
			console.ExpectString("xxx")
			console.SendLine("")
			time.Sleep(10 * time.Millisecond)
			console.ExpectString("xxx")
			console.SendLine(":wq!")
			console.ExpectEOF()
		}
		RunTest(t, procedure, testFn)
		assert.Equal(t, ":wq!", filename)
	})
	t.Run("empty with the override", func(t *testing.T) {
		var filename string
		testFn := func(stdio terminal.Stdio) error {
			var err error
			filename, err = FileSelector("xxx", "help", WithOutput(stdio), WithEmptyFilename("override"))
			return err
		}
		procedure := func(t *testing.T, console *expect.Console) {
			console.ExpectString("xxx")
			console.SendLine("")
			console.ExpectEOF()
		}
		RunTest(t, procedure, testFn)
		assert.Equal(t, "override", filename)
	})
}
