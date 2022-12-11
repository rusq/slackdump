package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
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
		procedure := func(t *testing.T, console console) {
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
		procedure := func(t *testing.T, c console) {
			c.ExpectString("xxx")
			c.SendLine("")
			time.Sleep(10 * time.Millisecond)
			c.ExpectString("xxx")
			c.SendLine(":wq!")
			c.ExpectEOF()
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
		procedure := func(t *testing.T, c console) {
			c.ExpectString("xxx")
			c.SendLine("")
			c.ExpectEOF()
		}
		RunTest(t, procedure, testFn)
		assert.Equal(t, "override", filename)
	})
	t.Run("overwrite", func(t *testing.T) {
		var filename string
		dir := t.TempDir()
		testfile := filepath.Join(dir, "testfile.txt")
		if err := os.WriteFile(testfile, []byte("unittest"), 0666); err != nil {
			t.Fatal(err)
		}
		RunTest(
			t,
			func(t *testing.T, c console) {
				c.ExpectString("make me overwrite this")
				c.SendLine(testfile)
				c.ExpectString("exists")
				c.SendLine("Y")
				c.ExpectEOF()
			},
			func(s terminal.Stdio) error {
				var err error
				filename, err = FileSelector("make me overwrite this", "", WithOutput(s))
				return err
			})
		assert.Equal(t, testfile, filename)
	})
}
