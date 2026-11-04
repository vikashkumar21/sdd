package install

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vikashkumar21/sdd/internal/fsutil"
)

const (
	beginMarker = "# sdd:begin path"
	endMarker   = "# sdd:end path"
)

// InstallOptions holds parameters for installing sdd to the user's system.
type InstallOptions struct {
	ExecutablePath string
	Home           func() string
	Getenv         func(string) string
	GOOS           string
	Force          bool
	DryRun         bool
}

// InstallResult contains the outcome of an installation.
type InstallResult struct {
	BinaryPath       string
	TargetDir        string
	ProfilePaths     []string
	AlreadyInstalled bool
	PathUpdated      bool
}

// ResolveTarget returns the target binary path and directory for the given OS and environment.
func ResolveTarget(goos string, home string, getenv func(string) string) (binaryPath, targetDir string, err error) {
	if goos == "windows" {
		localAppData := ""
		if getenv != nil {
			localAppData = getenv("LOCALAPPDATA")
		}
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		targetDir = filepath.Join(localAppData, "Programs", "sdd")
		binaryPath = filepath.Join(targetDir, "sdd.exe")
		return binaryPath, targetDir, nil
	}

	if home == "" {
		return "", "", errors.New("cannot determine user home directory")
	}
	targetDir = filepath.Join(home, ".local", "bin")
	binaryPath = filepath.Join(targetDir, "sdd")
	return binaryPath, targetDir, nil
}

// Install copies the running executable to the user binary directory and configures PATH.
func Install(opts InstallOptions) (InstallResult, error) {
	var res InstallResult

	goos := opts.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}

	home := ""
	if opts.Home != nil {
		home = opts.Home()
	} else {
		home, _ = os.UserHomeDir()
	}

	sourcePath := opts.ExecutablePath
	if sourcePath == "" {
		execPath, err := os.Executable()
		if err != nil {
			return res, fmt.Errorf("determine executable path: %w", err)
		}
		sourcePath = execPath
	}
	sourcePath = filepath.Clean(sourcePath)

	targetBinary, targetDir, err := ResolveTarget(goos, home, opts.Getenv)
	if err != nil {
		return res, err
	}
	res.BinaryPath = targetBinary
	res.TargetDir = targetDir

	if !opts.Force && sourcePath == targetBinary {
		res.AlreadyInstalled = true
		return res, nil
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return res, fmt.Errorf("read source binary %s: %w", sourcePath, err)
	}

	if !opts.DryRun {
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return res, fmt.Errorf("create directory %s: %w", targetDir, err)
		}
		if err := fsutil.WriteAtomic(targetBinary, data, 0o755); err != nil {
			return res, fmt.Errorf("write binary %s: %w", targetBinary, err)
		}
	}

	if goos == "windows" {
		currentPath := ""
		if opts.Getenv != nil {
			currentPath = opts.Getenv("PATH")
		} else {
			currentPath = os.Getenv("PATH")
		}
		if !strings.Contains(strings.ToLower(currentPath), strings.ToLower(targetDir)) {
			if !opts.DryRun {
				// Update user-level PATH via setx (standard Windows tool, no admin needed)
				_ = exec.Command("setx", "PATH", currentPath+";"+targetDir).Run()
			}
			res.PathUpdated = true
		}
		return res, nil
	}

	// Unix shell profile updates
	profiles := detectShellProfiles(home, goos)
	pathEntry := fmt.Sprintf("%s\nexport PATH=\"$HOME/.local/bin:$PATH\"\n%s\n", beginMarker, endMarker)

	for _, profile := range profiles {
		content, readErr := os.ReadFile(profile)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				// Only create if it's the primary shell profile
				if profile == filepath.Join(home, ".zshrc") || profile == filepath.Join(home, ".bashrc") {
					if !opts.DryRun {
						if err := fsutil.WriteAtomic(profile, []byte(pathEntry), 0o644); err == nil {
							res.ProfilePaths = append(res.ProfilePaths, profile)
							res.PathUpdated = true
						}
					} else {
						res.ProfilePaths = append(res.ProfilePaths, profile)
						res.PathUpdated = true
					}
				}
				continue
			}
			continue
		}

		s := string(content)
		if strings.Contains(s, beginMarker) || strings.Contains(s, "$HOME/.local/bin") || strings.Contains(s, "${HOME}/.local/bin") || strings.Contains(s, ".local/bin") {
			// Already configured
			continue
		}

		newContent := s
		if !strings.HasSuffix(newContent, "\n") && len(newContent) > 0 {
			newContent += "\n"
		}
		newContent += "\n" + pathEntry

		if !opts.DryRun {
			if err := fsutil.WriteAtomic(profile, []byte(newContent), 0o644); err != nil {
				return res, fmt.Errorf("update shell profile %s: %w", profile, err)
			}
		}
		res.ProfilePaths = append(res.ProfilePaths, profile)
		res.PathUpdated = true
	}

	return res, nil
}

func detectShellProfiles(home string, goos string) []string {
	var candidates []string
	if goos == "darwin" {
		candidates = append(candidates, filepath.Join(home, ".zshrc"), filepath.Join(home, ".bash_profile"))
	} else {
		candidates = append(candidates, filepath.Join(home, ".bashrc"), filepath.Join(home, ".zshrc"), filepath.Join(home, ".profile"))
	}
	return candidates
}

// UninstallOptions holds parameters for uninstalling sdd.
type UninstallOptions struct {
	Home   func() string
	Getenv func(string) string
	GOOS   string
	DryRun bool
}

// UninstallResult contains the outcome of an uninstallation.
type UninstallResult struct {
	BinaryPath   string
	Removed      bool
	CleanedFiles []string
}

// Uninstall removes the installed binary and cleans up shell profile PATH entries.
func Uninstall(opts UninstallOptions) (UninstallResult, error) {
	var res UninstallResult

	goos := opts.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}

	home := ""
	if opts.Home != nil {
		home = opts.Home()
	} else {
		home, _ = os.UserHomeDir()
	}

	targetBinary, _, err := ResolveTarget(goos, home, opts.Getenv)
	if err != nil {
		return res, err
	}
	res.BinaryPath = targetBinary

	if _, err := os.Stat(targetBinary); err == nil {
		if !opts.DryRun {
			if err := os.Remove(targetBinary); err != nil {
				return res, fmt.Errorf("remove binary %s: %w", targetBinary, err)
			}
		}
		res.Removed = true
	}

	if goos != "windows" {
		profiles := detectShellProfiles(home, goos)
		for _, profile := range profiles {
			data, err := os.ReadFile(profile)
			if err != nil {
				continue
			}
			content := string(data)
			if !strings.Contains(content, beginMarker) {
				continue
			}
			cleaned := removeMarkedBlock(content, beginMarker, endMarker)
			if !opts.DryRun {
				if err := fsutil.WriteAtomic(profile, []byte(cleaned), 0o644); err != nil {
					return res, fmt.Errorf("clean profile %s: %w", profile, err)
				}
			}
			res.CleanedFiles = append(res.CleanedFiles, profile)
		}
	}

	return res, nil
}

func removeMarkedBlock(content, begin, end string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var out []string
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == begin {
			inBlock = true
			continue
		}
		if inBlock {
			if trimmed == end {
				inBlock = false
			}
			continue
		}
		out = append(out, line)
	}

	res := strings.Join(out, "\n")
	if strings.HasSuffix(content, "\n") && !strings.HasSuffix(res, "\n") && len(res) > 0 {
		res += "\n"
	}
	return res
}
