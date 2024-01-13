package info

import (
	"io/fs"
	"os"
	"strings"
)

type sysinfo struct {
	OS         osinfo    `json:"os"`
	Workspace  workspace `json:"workspace"`
	Playwright pwinfo    `json:"playwright"`
	Rod        rodinfo   `json:"rod"`
	EzLogin    EZLogin   `json:"ez_login"`
}

func collect() *sysinfo {
	var si = new(sysinfo)
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
	return si

}

const (
	home = "$HOME"
)

var replaceFn = strings.NewReplacer(should(os.UserHomeDir()), home).Replace

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
	return "*ERROR: " + replaceFn(err.Error()) + "*"
}
