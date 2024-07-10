package pwcompat

import "path/filepath"

func init() {
	cacheDir = filepath.Join(homedir, ".cache")
	nodeExe = "node"
}
