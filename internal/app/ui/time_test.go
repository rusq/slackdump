package ui

import (
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/stretchr/testify/assert"
)

func TestTime(t *testing.T) {
	t.Run("valid date", func(t *testing.T) {
		var tm time.Time
		testFn := func(stdio terminal.Stdio) error {
			var err error
			tm, err = Time("UNIT", WithOutput(stdio))
			return err
		}
		procedure := func(t *testing.T, console console) {
			console.ExpectString("UNIT date")
			console.SendLine("2019-09-16")
			console.ExpectString("UNIT time")
			console.SendLine("15:16:17")
			console.ExpectEOF()
		}
		RunTest(t, procedure, testFn)
		assert.Equal(t, time.Date(2019, 9, 16, 15, 16, 17, 0, time.UTC), tm)
	})
	t.Run("invalid date", func(t *testing.T) {
		var tm time.Time
		testFn := func(stdio terminal.Stdio) error {
			var err error
			tm, err = Time("UNIT", WithOutput(stdio))
			return err
		}
		procedure := func(t *testing.T, console console) {
			console.ExpectString("UNIT date")
			console.SendLine("2")
			console.ExpectString(`invalid input, expected date format: YYYY-MM-DD`)
			console.SendLine("2019-09-16")
			console.ExpectString("UNIT time")
			console.SendLine("15:16:17")
			console.ExpectEOF()
		}
		RunTest(t, procedure, testFn)
		assert.Equal(t, time.Date(2019, 9, 16, 15, 16, 17, 0, time.UTC), tm)
	})
	t.Run("invalid time", func(t *testing.T) {
		var tm time.Time
		testFn := func(stdio terminal.Stdio) error {
			var err error
			tm, err = Time("UNIT", WithOutput(stdio))
			return err
		}
		procedure := func(t *testing.T, console console) {
			console.ExpectString("UNIT date")
			console.SendLine("2019-09-16")
			console.ExpectString("UNIT time")
			console.SendLine("15")
			console.ExpectString(`invalid input, expected time format: HH:MM:SS`)
			console.ExpectString("UNIT time")
			console.SendLine("15:16:17")
			console.ExpectEOF()
		}
		RunTest(t, procedure, testFn)
		assert.Equal(t, time.Date(2019, 9, 16, 15, 16, 17, 0, time.UTC), tm)
	})
}
