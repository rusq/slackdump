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

package updater

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/updater/github"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/updater/platform"
)

var (
	ErrUpdateFailed        = errors.New("update failed")
	ErrDownloadFailed      = errors.New("download failed")
	ErrPlatformNotDetected = errors.New("platform not detected")
)

// AutoUpdate attempts to update slackdump to the latest version.
func (u Updater) AutoUpdate(ctx context.Context, latest Release) error {
	// Detect platform
	p, err := platform.Detect(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPlatformNotDetected, err)
	}

	slog.InfoContext(ctx, "Detected platform", "os", p.OS, "arch", p.Arch, "package_system", p.PackageSystem.String(), "exe_path", p.ExePath)

	switch p.PackageSystem {
	case platform.Homebrew:
		return u.updateHomebrew(ctx, latest)
	case platform.Pacman:
		return u.updatePacman(ctx, latest)
	case platform.APT:
		return u.updateAPT(ctx, latest)
	case platform.Binary:
		return u.updateBinary(ctx, latest, p)
	default:
		return fmt.Errorf("%w: unknown package system", ErrUpdateFailed)
	}
}

// updateHomebrew updates slackdump using Homebrew.
func (u Updater) updateHomebrew(ctx context.Context, latest Release) error {
	slog.InfoContext(ctx, "Updating via Homebrew...")

	// First, update brew to ensure we have the latest formula
	slog.InfoContext(ctx, "Updating Homebrew...")
	cmd := exec.CommandContext(ctx, "brew", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update homebrew: %w", err)
	}

	// Check if brew has the latest version
	outdated, brewVersion, err := platform.IsBrewOutdated(ctx, latest.Version)
	if err != nil {
		slog.WarnContext(ctx, "Failed to check brew version, proceeding with upgrade", "error", err)
	} else if outdated {
		slog.WarnContext(ctx, "Homebrew formula is behind GitHub release", "brew_version", brewVersion, "github_version", latest.Version)
		slog.InfoContext(ctx, "The update will proceed, but you may not get the latest version immediately")
		slog.InfoContext(ctx, "If the formula is not updated after this, please wait for the Homebrew maintainers to update it")
	}

	// Upgrade slackdump
	slog.InfoContext(ctx, "Upgrading slackdump...")
	cmd = exec.CommandContext(ctx, "brew", "upgrade", "slackdump")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Check if it's already up to date
		if strings.Contains(err.Error(), "already installed") {
			slog.InfoContext(ctx, "Slackdump is already at the latest version available in Homebrew")
			return nil
		}
		return fmt.Errorf("%w: brew upgrade failed: %v", ErrUpdateFailed, err)
	}

	return nil
}

// updatePacman updates slackdump using Pacman (Arch Linux package manager).
func (u Updater) updatePacman(ctx context.Context, latest Release) error {
	slog.InfoContext(ctx, "Updating via Pacman...")

	// Update package database and upgrade slackdump in one command
	slog.InfoContext(ctx, "Syncing package database and upgrading slackdump...")
	cmd := exec.CommandContext(ctx, "sudo", "pacman", "-Sy", "--noconfirm", "slackdump")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: pacman upgrade failed: %v", ErrUpdateFailed, err)
	}

	return nil
}

// updateAPT updates slackdump using APT (Debian/Ubuntu package manager).
func (u Updater) updateAPT(ctx context.Context, latest Release) error {
	slog.InfoContext(ctx, "Updating via APT...")

	// Update package index
	slog.InfoContext(ctx, "Updating APT index...")
	cmd := exec.CommandContext(ctx, "sudo", "apt", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update apt index: %w", err)
	}

	// Upgrade slackdump
	slog.InfoContext(ctx, "Upgrading slackdump...")
	cmd = exec.CommandContext(ctx, "sudo", "apt", "install", "--only-upgrade", "slackdump")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: apt upgrade failed: %v", ErrUpdateFailed, err)
	}

	return nil
}

// updateBinary updates slackdump by downloading the binary from GitHub.
func (u Updater) updateBinary(ctx context.Context, latest Release, p *platform.Platform) error {
	slog.InfoContext(ctx, "Updating via binary download from GitHub...")

	// Get the latest release info to find the correct asset
	rel, err := u.cl.Latest(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	// Find the correct asset for this platform
	asset, err := findAsset(rel, p.OS, p.Arch)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "Downloading asset", "name", asset.Name, "size", asset.Size, "url", asset.BrowserDownloadURL)

	// Download the asset
	downloadPath, err := downloadAsset(ctx, asset)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(downloadPath); err != nil {
			slog.WarnContext(ctx, "Failed to remove download file", "path", downloadPath, "err", err)
		}
	}()

	// Extract and replace the binary
	if err := replaceBinary(ctx, downloadPath, p.ExePath, p.OS); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Binary updated successfully")
	return nil
}

// findAsset finds the correct asset for the given platform.
func findAsset(rel *github.Release, osName, arch string) (*github.Asset, error) {
	// Map Go OS/arch names to asset naming conventions
	osMap := map[string]string{
		"darwin":  "Darwin",
		"linux":   "Linux",
		"windows": "Windows",
	}
	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
		"386":   "i386",
	}

	osStr, ok := osMap[osName]
	if !ok {
		return nil, fmt.Errorf("unsupported OS: %s", osName)
	}

	archStr, ok := archMap[arch]
	if !ok {
		return nil, fmt.Errorf("unsupported architecture: %s", arch)
	}

	// Look for the asset that matches our platform
	// Asset names typically follow pattern: slackdump_<version>_<OS>_<arch>.zip
	for _, asset := range rel.Assets {
		name := asset.Name
		// Check if the asset name contains the OS and architecture
		if strings.Contains(name, osStr) && strings.Contains(name, archStr) {
			return &asset, nil
		}
	}

	return nil, fmt.Errorf("no asset found for platform: %s/%s", osName, arch)
}

// downloadAsset downloads an asset from GitHub.
func downloadAsset(ctx context.Context, asset *github.Asset) (string, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "slackdump-update-*.zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			slog.WarnContext(ctx, "Failed to close temp file", "path", tmpFile.Name(), "err", err)
		}
	}()

	// Download the asset
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "Failed to close response body", "err", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status code %d", ErrDownloadFailed, resp.StatusCode)
	}

	// Write to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write download: %w", err)
	}

	return tmpFile.Name(), nil
}

// replaceBinary extracts the binary from the zip and replaces the current executable.
func replaceBinary(ctx context.Context, zipPath, exePath, osName string) error {
	// Open the zip file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			slog.WarnContext(ctx, "Failed to close zip file", "path", zipPath, "err", err)
		}
	}()

	// Find the slackdump binary in the zip
	var binaryFile *zip.File
	binaryName := "slackdump"
	if osName == "windows" {
		binaryName = "slackdump.exe"
	}

	for _, f := range r.File {
		// Check if the base name matches the binary name
		if filepath.Base(f.Name) == binaryName {
			binaryFile = f
			break
		}
	}

	if binaryFile == nil {
		return fmt.Errorf("binary not found in zip archive")
	}

	// Extract the binary to a temporary location
	tmpBinary, err := os.CreateTemp("", "slackdump-new-*")
	if err != nil {
		return fmt.Errorf("failed to create temp binary: %w", err)
	}
	tmpBinaryPath := tmpBinary.Name()
	defer func() {
		if err := os.Remove(tmpBinaryPath); err != nil && !os.IsNotExist(err) {
			slog.WarnContext(ctx, "Failed to remove temp binary", "path", tmpBinaryPath, "err", err)
		}
	}()

	rc, err := binaryFile.Open()
	if err != nil {
		return fmt.Errorf("failed to open binary from zip: %w", err)
	}
	defer func() {
		if err := rc.Close(); err != nil {
			slog.WarnContext(ctx, "Failed to close binary file from zip", "err", err)
		}
	}()

	if _, err := io.Copy(tmpBinary, rc); err != nil {
		if closeErr := tmpBinary.Close(); closeErr != nil {
			slog.WarnContext(ctx, "Failed to close temp binary after copy error", "err", closeErr)
		}
		return fmt.Errorf("failed to extract binary: %w", err)
	}
	if err := tmpBinary.Close(); err != nil {
		return fmt.Errorf("failed to close temp binary: %w", err)
	}

	// Make the new binary executable (Unix-like systems)
	if osName != "windows" {
		if err := os.Chmod(tmpBinaryPath, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	// Backup the current binary
	backupPath := exePath + ".bak"
	if err := os.Rename(exePath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	// Move the new binary to the executable path
	if err := os.Rename(tmpBinaryPath, exePath); err != nil {
		// Try to restore the backup
		if restoreErr := os.Rename(backupPath, exePath); restoreErr != nil {
			return fmt.Errorf("failed to replace binary: %w (failed to restore backup: %v)", err, restoreErr)
		}
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Remove the backup; log a warning if cleanup fails.
	if err := os.Remove(backupPath); err != nil {
		slog.WarnContext(ctx, "Failed to remove backup binary", "path", backupPath, "err", err)
	}

	slog.InfoContext(ctx, "Binary replaced successfully", "path", exePath)
	return nil
}
