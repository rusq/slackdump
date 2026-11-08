package repository

import "strings"

// Open modes for [DSN].
const (
	ModeDefault = ""   // read-write, create if not exists
	ModeRO      = "ro" // read-only
	ModeRW      = "rw" // read-write, must exist
)

// connPragmas are applied by the driver to every new connection in the pool.
// Connection-scoped pragmas (foreign_keys, synchronous) must be set this way:
// executing them through *sql.DB affects only the one pooled connection that
// happened to run the statement.
var connPragmas = []string{
	"foreign_keys(1)",
	"synchronous(NORMAL)",
}

// uriPathEscaper escapes characters that have a special meaning in SQLite URI
// filenames.  '%' must be first.
var uriPathEscaper = strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23")

// DSN returns the data source name for the SQLite database at path, opened in
// the given mode (one of ModeDefault, ModeRO or ModeRW).  The DSN enables
// foreign keys and sets other connection pragmas on every pooled connection.
// Always use it to open slackdump databases.
func DSN(path string, mode string) string {
	var buf strings.Builder
	buf.WriteString("file:")
	buf.WriteString(uriPathEscaper.Replace(path))
	sep := byte('?')
	if mode != "" {
		buf.WriteByte(sep)
		buf.WriteString("mode=")
		buf.WriteString(mode)
		sep = '&'
	}
	for _, p := range connPragmas {
		buf.WriteByte(sep)
		buf.WriteString("_pragma=")
		buf.WriteString(p)
		sep = '&'
	}
	return buf.String()
}
