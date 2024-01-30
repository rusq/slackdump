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
	"io/fs"
	"os"
	"strings"
)

type SysInfo struct {
	OS         OSInfo    `json:"os"`
	Workspace  Workspace `json:"workspace"`
	Playwright PwInfo    `json:"playwright"`
	Rod        RodInfo   `json:"rod"`
	EzLogin    EZLogin   `json:"ez_login"`
}

// Collect collects system information, replacing user's name in paths with
// "$HOME".
func Collect() *SysInfo {
	return collect(homeReplacer)
}

// CollectWithPathReplacer collects the informaiton with the custom path
// replacing function.
func CollectRaw() *SysInfo {
	return collect(noopReplacer)
}

func collect(fn func(string) string) *SysInfo {
	si := new(SysInfo)
	collectors := []func(PathReplFunc){
		si.Workspace.collect,
		si.Playwright.collect,
		si.Rod.collect,
		si.EzLogin.collect,
		si.OS.collect,
	}
	for _, c := range collectors {
		c(fn)
	}
	return si
}

const (
	home = "$HOME"
)

// PathReplFunc is the signature of the function that replaces paths.
type PathReplFunc func(string) string

var (
	homeReplacer = strings.NewReplacer(should(os.UserHomeDir()), home).Replace
	noopReplacer = func(s string) string { return s }
)

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

func loser(err error) string {
	return "*ERROR: " + homeReplacer(err.Error()) + "*"
}
