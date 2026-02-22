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
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	ErrChecksumMismatch    = errors.New("checksum mismatch")
	ErrChecksumNotFound    = errors.New("checksum not found")
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
	slog.WarnContext(ctx, "This will run a privileged command with sudo")
	slog.InfoContext(ctx, "Command to execute", "cmd", "sudo pacman -Sy --noconfirm slackdump")
	slog.InfoContext(ctx, "Note: --noconfirm flag will auto-approve the installation without prompting")

	// Update package database and upgrade slackdump in one command
	// Note: We use --noconfirm because the user explicitly requested automatic update via -auto flag
	// The command only targets the specific 'slackdump' package, not a full system upgrade
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
	slog.WarnContext(ctx, "This will run privileged commands with sudo")

	// Update package index
	slog.InfoContext(ctx, "Command to execute", "cmd", "sudo apt update")
	slog.InfoContext(ctx, "Updating APT index...")
	cmd := exec.CommandContext(ctx, "sudo", "apt", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update apt index: %w", err)
	}

	// Upgrade slackdump
	// Note: --only-upgrade ensures we only update slackdump if it's already installed
	// APT will still prompt for confirmation unless run non-interactively
	slog.InfoContext(ctx, "Command to execute", "cmd", "sudo apt install --only-upgrade slackdump")
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
	downloadPath, err := downloadAsset(ctx, rel, asset)
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

// downloadAsset downloads an asset from GitHub and verifies its SHA256 checksum.
func downloadAsset(ctx context.Context, rel *github.Release, asset *github.Asset) (string, error) {
	// Determine file extension from asset name
	ext := filepath.Ext(asset.Name)
	if ext == ".gz" && strings.HasSuffix(asset.Name, ".tar.gz") {
		ext = ".tar.gz"
	}

	// Create a temporary file with appropriate extension
	tmpFile, err := os.CreateTemp("", "slackdump-update-*"+ext)
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

	// Calculate SHA256 hash while writing to temp file
	hash := sha256.New()
	multiWriter := io.MultiWriter(tmpFile, hash)

	if _, err := io.Copy(multiWriter, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write download: %w", err)
	}

	// Verify checksum if available
	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	expectedHash, err := getExpectedChecksum(ctx, rel, asset.Name)
	if err != nil {
		// Log warning but don't fail if checksums file is not available
		slog.WarnContext(ctx, "Could not verify checksum", "err", err, "file", asset.Name)
	} else {
		slog.InfoContext(ctx, "Verifying checksum", "file", asset.Name, "expected", expectedHash, "calculated", calculatedHash)
		if calculatedHash != expectedHash {
			return "", fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expectedHash, calculatedHash)
		}
		slog.InfoContext(ctx, "Checksum verification passed", "file", asset.Name)
	}

	return tmpFile.Name(), nil
}

// getExpectedChecksum downloads and parses the checksums.txt file to find the expected hash for the given asset.
func getExpectedChecksum(ctx context.Context, rel *github.Release, assetName string) (string, error) {
	// Find the checksums.txt asset
	var checksumAsset *github.Asset
	for _, asset := range rel.Assets {
		if asset.Name == "checksums.txt" {
			checksumAsset = &asset
			break
		}
	}

	if checksumAsset == nil {
		return "", fmt.Errorf("%w: checksums.txt not found in release", ErrChecksumNotFound)
	}

	// Download checksums.txt
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumAsset.BrowserDownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for checksums: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download checksums: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "Failed to close checksums response body", "err", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download checksums: status code %d", resp.StatusCode)
	}

	// Parse checksums.txt to find the hash for our asset
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Format: <hash>  <filename>
		// Standard checksum format uses 2 spaces, but we handle 1+ spaces
		// Split on first occurrence of whitespace to handle filenames with spaces
		hash, filename, found := strings.Cut(line, " ")
		if !found {
			continue
		}
		// Trim any additional leading whitespace from filename
		filename = strings.TrimLeft(filename, " ")
		if filename == assetName {
			return hash, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading checksums: %w", err)
	}

	return "", fmt.Errorf("%w: no checksum found for %s", ErrChecksumNotFound, assetName)
}

// replaceBinary extracts the binary from the archive (tar.gz or zip) and replaces the current executable.
func replaceBinary(ctx context.Context, archivePath, exePath, osName string) error {
	binaryName := "slackdump"
	if osName == "windows" {
		binaryName = "slackdump.exe"
	}

	// Determine archive format from file extension
	var tmpBinaryPath string
	var err error

	if strings.HasSuffix(archivePath, ".tar.gz") {
		tmpBinaryPath, err = extractFromTarGz(ctx, archivePath, binaryName)
	} else if strings.HasSuffix(archivePath, ".zip") {
		tmpBinaryPath, err = extractFromZip(ctx, archivePath, binaryName)
	} else {
		return fmt.Errorf("unsupported archive format: %s (expected .tar.gz or .zip)", archivePath)
	}

	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(tmpBinaryPath); err != nil && !os.IsNotExist(err) {
			slog.WarnContext(ctx, "Failed to remove temp binary", "path", tmpBinaryPath, "err", err)
		}
	}()

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

// extractFromTarGz extracts a binary from a tar.gz archive and returns the path to the extracted file.
func extractFromTarGz(ctx context.Context, archivePath, binaryName string) (string, error) {
	// Open the tar.gz file
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.WarnContext(ctx, "Failed to close archive file", "path", archivePath, "err", err)
		}
	}()

	// Create gzip reader
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if err := gzr.Close(); err != nil {
			slog.WarnContext(ctx, "Failed to close gzip reader", "err", err)
		}
	}()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Find the binary in the tar archive
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Check if this is the binary we're looking for
		if filepath.Base(header.Name) == binaryName && header.Typeflag == tar.TypeReg {
			// Extract to temporary file
			tmpFile, err := os.CreateTemp("", "slackdump-new-*")
			if err != nil {
				return "", fmt.Errorf("failed to create temp file: %w", err)
			}
			tmpPath := tmpFile.Name()

			if _, err := io.Copy(tmpFile, tr); err != nil {
				_ = tmpFile.Close()
				_ = os.Remove(tmpPath)
				return "", fmt.Errorf("failed to extract binary: %w", err)
			}

			if err := tmpFile.Close(); err != nil {
				_ = os.Remove(tmpPath)
				return "", fmt.Errorf("failed to close temp file: %w", err)
			}

			return tmpPath, nil
		}
	}

	return "", fmt.Errorf("binary not found in tar.gz archive: %s", binaryName)
}

// extractFromZip extracts a binary from a zip archive and returns the path to the extracted file.
func extractFromZip(ctx context.Context, archivePath, binaryName string) (string, error) {
	// Open the zip file
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip: %w", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			slog.WarnContext(ctx, "Failed to close zip file", "path", archivePath, "err", err)
		}
	}()

	// Find the slackdump binary in the zip
	for _, f := range r.File {
		// Check if the base name matches the binary name
		if filepath.Base(f.Name) == binaryName {
			// Extract to temporary file
			tmpFile, err := os.CreateTemp("", "slackdump-new-*")
			if err != nil {
				return "", fmt.Errorf("failed to create temp file: %w", err)
			}
			tmpPath := tmpFile.Name()

			rc, err := f.Open()
			if err != nil {
				_ = tmpFile.Close()
				_ = os.Remove(tmpPath)
				return "", fmt.Errorf("failed to open file in zip: %w", err)
			}

			_, copyErr := io.Copy(tmpFile, rc)
			closeErr1 := rc.Close()
			closeErr2 := tmpFile.Close()

			if copyErr != nil {
				_ = os.Remove(tmpPath)
				return "", fmt.Errorf("failed to extract binary: %w", copyErr)
			}
			if closeErr1 != nil {
				slog.WarnContext(ctx, "Failed to close zip entry", "err", closeErr1)
			}
			if closeErr2 != nil {
				_ = os.Remove(tmpPath)
				return "", fmt.Errorf("failed to close temp file: %w", closeErr2)
			}

			return tmpPath, nil
		}
	}

	return "", fmt.Errorf("binary not found in zip archive: %s", binaryName)
}
