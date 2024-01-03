package ui

// func TestTime(t *testing.T) {
// 	t.Skip()
// 	t.Run("valid date", func(t *testing.T) {
// 		var tm time.Time
// 		testFn := func(stdio terminal.Stdio) error {
// 			var err error
// 			tm, err = Time("UNIT", WithOutput(stdio))
// 			return err
// 		}
// 		procedure := func(t *testing.T, console console) {
// 			console.ExpectString("UNIT date")
// 			console.SendLine("2019-09-16")
// 			console.ExpectString("UNIT time")
// 			console.SendLine("15:16:17")
// 			console.ExpectEOF()
// 		}
// 		RunTest(t, procedure, testFn)
// 		assert.Equal(t, time.Date(2019, 9, 16, 15, 16, 17, 0, time.UTC), tm)
// 	})
// 	t.Run("invalid date", func(t *testing.T) {
// 		var tm time.Time
// 		testFn := func(stdio terminal.Stdio) error {
// 			var err error
// 			tm, err = Time("UNIT", WithOutput(stdio))
// 			return err
// 		}
// 		procedure := func(t *testing.T, console console) {
// 			console.ExpectString("UNIT date")
// 			console.SendLine("2")
// 			console.ExpectString(`invalid input, expected date format: YYYY-MM-DD`)
// 			console.SendLine("2019-09-16")
// 			console.ExpectString("UNIT time")
// 			console.SendLine("15:16:17")
// 			console.ExpectEOF()
// 		}
// 		RunTest(t, procedure, testFn)
// 		assert.Equal(t, time.Date(2019, 9, 16, 15, 16, 17, 0, time.UTC), tm)
// 	})
// 	t.Run("invalid time", func(t *testing.T) {
// 		var tm time.Time
// 		testFn := func(stdio terminal.Stdio) error {
// 			var err error
// 			tm, err = Time("UNIT", WithOutput(stdio))
// 			return err
// 		}
// 		procedure := func(t *testing.T, console console) {
// 			console.ExpectString("UNIT date")
// 			console.SendLine("2019-09-16")
// 			console.ExpectString("UNIT time")
// 			console.SendLine("15")
// 			console.ExpectString(`invalid input, expected time format: HH:MM:SS`)
// 			console.ExpectString("UNIT time")
// 			console.SendLine("15:16:17")
// 			console.ExpectEOF()
// 		}
// 		RunTest(t, procedure, testFn)
// 		assert.Equal(t, time.Date(2019, 9, 16, 15, 16, 17, 0, time.UTC), tm)
// 	})
// 	t.Run("empty date", func(t *testing.T) {
// 		var tm time.Time
// 		var err error
// 		testFn := func(stdio terminal.Stdio) error {
// 			tm, err = Time("UNIT", WithOutput(stdio))
// 			return nil
// 		}
// 		procedure := func(t *testing.T, console console) {
// 			console.ExpectString("UNIT date")
// 			console.SendLine("")
// 			console.ExpectEOF()
// 		}
// 		RunTest(t, procedure, testFn)
// 		assert.Equal(t, time.Time{}, tm)
// 		assert.Equal(t, ErrEmptyOptionalInput, err)
// 	})
// 	t.Run("empty time does not pass validation", func(t *testing.T) {
// 		var tm time.Time
// 		var err error
// 		testFn := func(stdio terminal.Stdio) error {
// 			tm, err = Time("UNIT", WithOutput(stdio))
// 			return nil
// 		}
// 		procedure := func(t *testing.T, console console) {
// 			console.ExpectString("UNIT date")
// 			console.SendLine("2019-09-16")
// 			console.ExpectString("UNIT time")
// 			console.SendLine("")
// 			// validation fails, reenter
// 			console.ExpectString("invalid input, expected time format: HH:MM:SS")
// 			console.SendLine("15:16:17")
// 			console.ExpectEOF()
// 		}
// 		RunTest(t, procedure, testFn)
// 		assert.Equal(t, time.Date(2019, 9, 16, 15, 16, 17, 0, time.UTC), tm)
// 		assert.Nil(t, err)
// 	})
// }
