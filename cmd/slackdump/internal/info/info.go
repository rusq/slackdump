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
	"io/fs"
	"os"
	"strings"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/cache"
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

type sysinfo struct {
	OS         osinfo    `json:"os"`
	Workspace  workspace `json:"workspace"`
	Playwright pwinfo    `json:"playwright"`
	Rod        rodinfo   `json:"rod"`
	EzLogin    ezlogin   `json:"ez_login"`
}

type ezlogin struct {
	Flags   map[string]bool `json:"flags"`
	Browser string          `json:"browser"`
}

func (inf *ezlogin) collect() {
	inf.Flags = cache.EzLoginFlags()
	inf.Browser = cfg.Browser.String()
}

func runInfo(ctx context.Context, cmd *base.Command, args []string) error {
	var si sysinfo
	var collectors = []func(){
		si.Workspace.collect,
		si.Playwright.collect,
		si.Rod.collect,
		si.EzLogin.collect,
		si.OS.collect,
	}
	for _, c := range collectors {
		c()
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(si); err != nil {
		return err
	}

	return nil
}

var homerepl = strings.NewReplacer(should(os.UserHomeDir()), "~").Replace

func should(v string, err error) string {
	if err != nil {
		return "$$$ERROR$$$"
	}
	return v
}

func dirnames(des []fs.DirEntry) []string {
	var res []string
	for _, de := range des {
		if de.IsDir() && !strings.HasPrefix(de.Name(), ".") {
			res = append(res, de.Name())
		}
	}
	return res
}

func looser(err error) string {
	return "*ERROR: " + homerepl(err.Error()) + "*"
}
