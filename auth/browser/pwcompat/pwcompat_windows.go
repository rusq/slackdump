package pwcompat

import "path/filepath"

func init() {
	cacheDir = filepath.Join(homedir, "AppData", "Local")
	NodeExe = "node.exe"
}
