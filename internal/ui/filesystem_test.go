package ui

// func init() {
// 	// disable color output for all prompts to simplify testing
// 	core.DisableColor = true
// }

// func TestFileselector(t *testing.T) {
// 	t.Skip()
// 	t.Run("filename set", func(t *testing.T) {
// 		var filename string
// 		testFn := func(stdio terminal.Stdio) error {
// 			var err error
// 			filename, err = FileSelector("xxx", "help", WithOutput(stdio))
// 			return err
// 		}
// 		procedure := func(t *testing.T, console console) {
// 			console.ExpectString("xxx")
// 			console.SendLine("test.txt")
// 			console.ExpectEOF()
// 		}
// 		RunTest(t, procedure, testFn)
// 		assert.Equal(t, "test.txt", filename)
// 	})
// 	t.Run("empty, default not set", func(t *testing.T) {
// 		var filename string
// 		testFn := func(stdio terminal.Stdio) error {
// 			var err error
// 			filename, err = FileSelector("xxx", "help", WithOutput(stdio))
// 			return err
// 		}
// 		procedure := func(t *testing.T, c console) {
// 			c.ExpectString("xxx")
// 			c.SendLine("")
// 			time.Sleep(10 * time.Millisecond)
// 			c.ExpectString("xxx")
// 			c.SendLine(":wq!")
// 			c.ExpectEOF()
// 		}
// 		RunTest(t, procedure, testFn)
// 		assert.Equal(t, ":wq!", filename)
// 	})
// 	t.Run("empty, default set", func(t *testing.T) {
// 		var filename string
// 		testFn := func(stdio terminal.Stdio) error {
// 			var err error
// 			filename, err = FileSelector("xxx", "help", WithOutput(stdio), WithDefaultFilename("default_filename"))
// 			return err
// 		}
// 		procedure := func(t *testing.T, c console) {
// 			c.ExpectString("xxx")
// 			c.SendLine("")
// 			c.ExpectEOF()
// 		}
// 		RunTest(t, procedure, testFn)
// 		assert.Equal(t, "default_filename", filename)
// 	})
// 	t.Run("filename set and exist", func(t *testing.T) {
// 		var filename string
// 		dir := t.TempDir()
// 		testfile := filepath.Join(dir, "testfile.txt")
// 		if err := os.WriteFile(testfile, []byte("unittest"), 0666); err != nil {
// 			t.Fatal(err)
// 		}
// 		RunTest(
// 			t,
// 			func(t *testing.T, c console) {
// 				c.ExpectString("make me overwrite this")
// 				c.SendLine(testfile)
// 				c.ExpectString("exists")
// 				c.SendLine("Y")
// 				c.ExpectEOF()
// 			},
// 			func(s terminal.Stdio) error {
// 				var err error
// 				filename, err = FileSelector("make me overwrite this", "", WithOutput(s))
// 				return err
// 			})
// 		assert.Equal(t, testfile, filename)
// 	})
// 	t.Run("filename set, default empty, must exist set", func(t *testing.T) {
// 		var filename string
// 		dir := t.TempDir()
// 		testfile := filepath.Join(dir, "testfile.txt")
// 		if err := os.WriteFile(testfile, []byte("unittest"), 0666); err != nil {
// 			t.Fatal(err)
// 		}
// 		RunTest(
// 			t,
// 			func(t *testing.T, c console) {
// 				// attempt 1.
// 				c.ExpectString("non-existing")
// 				c.SendLine(filepath.Join(dir, "$$$$.XXX"))
// 				c.ExpectString("file must exist")
// 				// attempt 2.
// 				c.ExpectString("non-existing")
// 				c.SendLine(testfile)
// 				c.ExpectEOF()
// 			},
// 			func(s terminal.Stdio) error {
// 				var err error
// 				filename, err = FileSelector("non-existing test", "", WithOutput(s), WithMustExist(true))
// 				return err
// 			})
// 		assert.Equal(t, testfile, filename)
// 	})
// 	t.Run("filename not set, default set, must exist set", func(t *testing.T) {
// 		var filename string
// 		dir := t.TempDir()
// 		testfile := filepath.Join(dir, "testfile.txt")
// 		if err := os.WriteFile(testfile, []byte("unittest"), 0666); err != nil {
// 			t.Fatal(err)
// 		}
// 		RunTest(
// 			t,
// 			func(t *testing.T, c console) {
// 				// attempt 1.
// 				c.ExpectString("variant 4")
// 				c.SendLine("")
// 				c.ExpectEOF()
// 			},
// 			func(s terminal.Stdio) error {
// 				var err error
// 				filename, err = FileSelector("variant 4", "", WithOutput(s), WithDefaultFilename(testfile), WithMustExist(true))
// 				return err
// 			})
// 		assert.Equal(t, testfile, filename)
// 	})
// }
