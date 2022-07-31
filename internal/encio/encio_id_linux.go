package encio

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"strings"

	"github.com/mzky/machineid"
)

const cgroupFile = "/proc/1/cgroup"

const (
	idLXC       = "lxc"
	idDocker    = "docker"
	idDockerNew = "0::/"
)

var machineIDFn = protectedIDwrapper

func protectedIDwrapper(appID string) (string, error) {
	if inContainer, err := isInContainer(cgroupFile); !inContainer || err != nil {
		return machineid.ProtectedID(appID)
	}
	compound := append(genID(), []byte(appID)...)
	id := sha256.Sum256(compound)
	return hex.EncodeToString(id[:]), nil
}

func genID() []byte {
	if id, err := os.Hostname(); err == nil {
		return []byte(id)
	}
	if buf, err := exec.Command("uname", "-n").Output(); err == nil {
		return buf
	}
	var b = make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return b
	}
	return []byte("no-machine-id")
}

// isInContainer checks if the service is being executed in docker or lxc
// container.
func isInContainer(cgroupPath string) (bool, error) {

	const maxlines = 5 // maximum lines to scan

	f, err := os.Open(cgroupPath)
	if err != nil {
		return false, err
	}
	defer f.Close()
	scan := bufio.NewScanner(f)

	lines := 0
	for scan.Scan() && !(lines > maxlines) {
		text := scan.Text()
		for _, s := range []string{idDockerNew, idDocker, idLXC} {
			if strings.Contains(text, s) {
				return true, nil
			}
		}
		lines++
	}
	if err := scan.Err(); err != nil {
		return false, err
	}

	return false, nil
}
