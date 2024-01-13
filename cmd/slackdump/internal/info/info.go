// Package info contains the Info command.  It collects some vital information
// about the environment, that could be useful for debugging some issues.
//
// # Collectors
//
// The information is collected by a set of collectors, each of which is
// responsible for collecting a specific piece of information.  The collectors
// are defined in separate files.
//
// A collector is a struct with a method "collect" defined on it.  The method
// collects the information and stores it in the struct.  The struct is then
// serialized to JSON and printed to stdout.
//
// The collect() method can choose not to terminate on error, but, for string
// methods, populate it with the error message, there's a function called
// "looser(err)" that returns an error with prefix "ERROR", the string will
// look like "*ERROR: some error*".
//
// Procedure to add new collectors:
//  1. Create a new file in this package, named after the collector.
//  2. Define a struct with the name of the collector, alternatively, if
//     it interfers with the imported package, add "*info" suffix, i.e.
//     "rodinfo"
//  3. Define a method "collect" on the struct, that collects the information
//     and populates struct fields.
//  4. Add the type to the "sysinfo" struct in this file.
//  5. Add the collector to the "collectors" slice in the "runInfo" function.
//
// Example of a collector:
//
//	type someinfo struct {
//	  SomeField string `json:"some_field"`
//	}
//
//	func (inf *someinfo) collect() {
//	  inf.SomeField = "some value"
//	}
package info

import (
	"context"
	"encoding/json"
	"os"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
)

// CmdInfo is the information command.
var CmdInfo = &base.Command{
	UsageLine: "slackdump info",
	Short:     "show information about slackdump environment",
	Run:       runInfo,
	Long: `# Info Command
	
**Info** shows information about Slackdump environment, such as local system paths, etc.
`,
}

func runInfo(ctx context.Context, cmd *base.Command, args []string) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(collect()); err != nil {
		return err
	}

	return nil
}
