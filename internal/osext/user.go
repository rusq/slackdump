package osext

import "os"

// IsRoot returns true if the process is running as root.
func IsRoot() bool {
	return os.Geteuid() == 0
}
