// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package platform provides OS and package manager detection functionality.
package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Platform represents the operating system and package manager.
type Platform struct {
	OS            string
	Arch          string
	PackageSystem PackageSystem
	ExePath       string
}

// PackageSystem represents different package management systems.
type PackageSystem int

const (
	Unknown PackageSystem = iota
	Homebrew
	APK    // Alpine Linux
	APT    // Debian/Ubuntu
	Binary // Direct binary installation (Windows, Linux fallback)
)

func (p PackageSystem) String() string {
	switch p {
	case Homebrew:
		return "homebrew"
	case APK:
		return "apk"
	case APT:
		return "apt"
	case Binary:
		return "binary"
	default:
		return "unknown"
	}
}

var (
	ErrUnsupportedPlatform = errors.New("unsupported platform")
	ErrExecutableNotFound  = errors.New("executable path not found")
)

// Detect detects the current platform and package manager.
func Detect(ctx context.Context) (*Platform, error) {
	p := &Platform{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// Find the executable path
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExecutableNotFound, err)
	}
	// Resolve symlinks
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable symlink: %w", err)
	}
	p.ExePath = exePath

	// Detect package manager
	switch p.OS {
	case "darwin":
		p.PackageSystem = detectMacOS(ctx, exePath)
	case "linux":
		p.PackageSystem = detectLinux(ctx, exePath)
	case "windows":
		p.PackageSystem = Binary
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, p.OS)
	}

	return p, nil
}

// detectMacOS detects if slackdump is installed via Homebrew on macOS.
func detectMacOS(ctx context.Context, exePath string) PackageSystem {
	// Check if the executable is in a Homebrew path
	if strings.Contains(exePath, "/homebrew/") || strings.Contains(exePath, "/Homebrew/") ||
		strings.Contains(exePath, "/usr/local/Cellar/") || strings.Contains(exePath, "/opt/homebrew/") {
		// Verify brew is available
		if _, err := exec.LookPath("brew"); err == nil {
			return Homebrew
		}
	}

	// Fallback to binary
	return Binary
}

// detectLinux detects the package manager on Linux.
func detectLinux(ctx context.Context, exePath string) PackageSystem {
	// Check for APK (Arch Linux)
	if _, err := exec.LookPath("apk"); err == nil {
		// Check if slackdump is installed via apk
		cmd := exec.CommandContext(ctx, "apk", "info", "slackdump")
		if err := cmd.Run(); err == nil {
			return APK
		}
	}

	// Check for APT (Debian/Ubuntu)
	if _, err := exec.LookPath("apt"); err == nil {
		// Check if slackdump is installed via apt
		cmd := exec.CommandContext(ctx, "dpkg", "-s", "slackdump")
		if err := cmd.Run(); err == nil {
			return APT
		}
	}

	// Fallback to binary
	return Binary
}

// IsBrewOutdated checks if Homebrew has an outdated version compared to GitHub.
func IsBrewOutdated(ctx context.Context, latestVersion string) (bool, string, error) {
	cmd := exec.CommandContext(ctx, "brew", "info", "--json=v2", "slackdump")
	output, err := cmd.Output()
	if err != nil {
		return false, "", fmt.Errorf("failed to get brew info: %w", err)
	}

	// Parse the JSON output to get the current formula version
	// This is a simplified version - in production, you'd want proper JSON parsing
	brewVersion := parseBrewVersion(string(output))
	if brewVersion == "" {
		return false, "", errors.New("failed to parse brew version")
	}

	// Compare versions (remove 'v' prefix if present)
	latestVersion = strings.TrimPrefix(latestVersion, "v")
	brewVersion = strings.TrimPrefix(brewVersion, "v")

	return brewVersion != latestVersion, brewVersion, nil
}

// parseBrewVersion extracts the version from brew info JSON output.
func parseBrewVersion(jsonOutput string) string {
	// Parse the JSON properly
	var result struct {
		Formulae []struct {
			Versions struct {
				Stable string `json:"stable"`
			} `json:"versions"`
		} `json:"formulae"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		return ""
	}

	if len(result.Formulae) == 0 {
		return ""
	}

	return result.Formulae[0].Versions.Stable
}
