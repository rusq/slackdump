package pwcompat

import "path/filepath"

func init() {
	cacheDir = filepath.Join(homedir, "Library", "Caches")
	NodeExe = "node"
}
