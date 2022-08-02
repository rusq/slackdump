package encio

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"strings"

	"github.com/denisbrodbeck/machineid"
)

const cgroupFile = "/proc/1/cgroup"

const (
	idLXC       = "lxc"
	idDocker    = "docker"
	idDockerNew = "0::/"
)

var machineIDFn = protectedIDwrapper

// protectedIDwrapper is a wrapper around machineid.ProtectedID.  If executed
// inside docker container, the machineid.ProtectedID will fail, because it
// relies on /etc/machine-id that may not be present.  If it fails to locate
// the machine ID it calls genID which will attempt to generate an ID using
// hostname, which is pretty random in docker, unless the user has assigned
// a specific name.
func protectedIDwrapper(appID string) (string, error) {
	if inContainer, err := isInContainer(cgroupFile); !inContainer || err != nil {
		return machineid.ProtectedID(appID)
	}
	compound := append(genID(), []byte(appID)...)
	id := sha256.Sum256(compound)
	return hex.EncodeToString(id[:]), nil
}

// genID generates an ID either from hostname or, if it is unable to get the
// hostname, it will return "no-machine-id"
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
