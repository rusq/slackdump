package pwcompat

import "path/filepath"

func init() {
	cacheDir = filepath.Join(homedir, "AppData", "Local")
	nodeExe = "node.exe"
}
