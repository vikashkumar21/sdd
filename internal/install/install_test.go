package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTarget(t *testing.T) {
	// Unix
	bin, dir, err := ResolveTarget("darwin", "/Users/alice", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedDir := filepath.Join("/Users/alice", ".local", "bin")
	expectedBin := filepath.Join(expectedDir, "sdd")
	if dir != expectedDir {
		t.Errorf("dir = %q, want %q", dir, expectedDir)
	}
	if bin != expectedBin {
		t.Errorf("bin = %q, want %q", bin, expectedBin)
	}

	// Linux
	bin, dir, err = ResolveTarget("linux", "/home/bob", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedDir = filepath.Join("/home/bob", ".local", "bin")
	expectedBin = filepath.Join(expectedDir, "sdd")
	if dir != expectedDir {
		t.Errorf("dir = %q, want %q", dir, expectedDir)
	}
	if bin != expectedBin {
		t.Errorf("bin = %q, want %q", bin, expectedBin)
	}

	// Windows with LOCALAPPDATA
	bin, dir, err = ResolveTarget("windows", "C:\\Users\\carol", func(k string) string {
		if k == "LOCALAPPDATA" {
			return "C:\\Users\\carol\\AppData\\Local"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedDir = filepath.Join("C:\\Users\\carol\\AppData\\Local", "Programs", "sdd")
	expectedBin = filepath.Join(expectedDir, "sdd.exe")
	if dir != expectedDir {
		t.Errorf("dir = %q, want %q", dir, expectedDir)
	}
	if bin != expectedBin {
		t.Errorf("bin = %q, want %q", bin, expectedBin)
	}

	// Empty home error
	_, _, err = ResolveTarget("darwin", "", nil)
	if err == nil {
		t.Errorf("expected error for empty home, got nil")
	}
}

func TestInstallUnix(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create fake source binary
	fakeSource := filepath.Join(tmpDir, "fake_sdd")
	if err := os.WriteFile(fakeSource, []byte("#!/bin/sh\necho test\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create pre-existing .zshrc
	zshrc := filepath.Join(homeDir, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("alias ll='ls -la'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := InstallOptions{
		ExecutablePath: fakeSource,
		Home:           func() string { return homeDir },
		GOOS:           "darwin",
	}

	// Run install
	res, err := Install(opts)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !res.PathUpdated {
		t.Errorf("expected PathUpdated = true")
	}

	// Verify installed binary
	expectedBin := filepath.Join(homeDir, ".local", "bin", "sdd")
	data, err := os.ReadFile(expectedBin)
	if err != nil {
		t.Fatalf("failed to read installed binary: %v", err)
	}
	if string(data) != "#!/bin/sh\necho test\n" {
		t.Errorf("installed binary content mismatch")
	}

	// Verify .zshrc content
	zshrcContent, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("failed to read .zshrc: %v", err)
	}
	if !strings.Contains(string(zshrcContent), beginMarker) {
		t.Errorf(".zshrc missing begin marker")
	}
	if !strings.Contains(string(zshrcContent), "export PATH=\"$HOME/.local/bin:$PATH\"") {
		t.Errorf(".zshrc missing export PATH")
	}
	if !strings.Contains(string(zshrcContent), "alias ll='ls -la'") {
		t.Errorf(".zshrc wiped existing content")
	}

	// Test idempotency: run Install again
	res2, err := Install(opts)
	if err != nil {
		t.Fatalf("Second install failed: %v", err)
	}
	if res2.PathUpdated {
		t.Errorf("expected PathUpdated = false on second run")
	}

	zshrcContent2, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(zshrcContent2), beginMarker) != 1 {
		t.Errorf("beginMarker duplicated in .zshrc: count = %d", strings.Count(string(zshrcContent2), beginMarker))
	}
}

func TestInstallDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fakeSource := filepath.Join(tmpDir, "fake_sdd")
	if err := os.WriteFile(fakeSource, []byte("echo binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	opts := InstallOptions{
		ExecutablePath: fakeSource,
		Home:           func() string { return homeDir },
		GOOS:           "linux",
		DryRun:         true,
	}

	res, err := Install(opts)
	if err != nil {
		t.Fatalf("Install dry-run failed: %v", err)
	}
	if !res.PathUpdated {
		t.Errorf("expected PathUpdated = true in dry run")
	}

	// Verify target binary was NOT created
	targetBin := filepath.Join(homeDir, ".local", "bin", "sdd")
	if _, err := os.Stat(targetBin); !os.IsNotExist(err) {
		t.Errorf("expected target binary not to exist in dry-run")
	}

	// Verify shell profile was NOT created
	bashrc := filepath.Join(homeDir, ".bashrc")
	if _, err := os.Stat(bashrc); !os.IsNotExist(err) {
		t.Errorf("expected .bashrc not to exist in dry-run")
	}
}

func TestInstallAlreadyInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	installedDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(installedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	installedBin := filepath.Join(installedDir, "sdd")
	if err := os.WriteFile(installedBin, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	opts := InstallOptions{
		ExecutablePath: installedBin,
		Home:           func() string { return homeDir },
		GOOS:           "darwin",
		Force:          false,
	}

	res, err := Install(opts)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if !res.AlreadyInstalled {
		t.Errorf("expected AlreadyInstalled = true")
	}
}

func TestUninstall(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	binDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}

	binPath := filepath.Join(binDir, "sdd")
	if err := os.WriteFile(binPath, []byte("sdd"), 0o755); err != nil {
		t.Fatal(err)
	}

	zshrc := filepath.Join(homeDir, ".zshrc")
	zshContent := "alias g=git\n# sdd:begin path\nexport PATH=\"$HOME/.local/bin:$PATH\"\n# sdd:end path\nalias k=kubectl\n"
	if err := os.WriteFile(zshrc, []byte(zshContent), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := UninstallOptions{
		Home: func() string { return homeDir },
		GOOS: "darwin",
	}

	res, err := Uninstall(opts)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if !res.Removed {
		t.Errorf("expected Removed = true")
	}
	if len(res.CleanedFiles) != 1 {
		t.Errorf("expected 1 cleaned file, got %d", len(res.CleanedFiles))
	}

	// Verify binary was removed
	if _, err := os.Stat(binPath); !os.IsNotExist(err) {
		t.Errorf("expected binary to be removed")
	}

	// Verify zshrc preserved user aliases but removed sdd block
	cleaned, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(cleaned)
	if strings.Contains(s, beginMarker) || strings.Contains(s, endMarker) {
		t.Errorf("markers still present in .zshrc: %s", s)
	}
	if !strings.Contains(s, "alias g=git") || !strings.Contains(s, "alias k=kubectl") {
		t.Errorf("user aliases were lost: %s", s)
	}
}

func TestRemoveMarkedBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "block in middle",
			input:    "header\n# sdd:begin path\nexport PATH=...\n# sdd:end path\nfooter\n",
			expected: "header\nfooter\n",
		},
		{
			name:     "block at start",
			input:    "# sdd:begin path\nexport PATH=...\n# sdd:end path\nfooter\n",
			expected: "footer\n",
		},
		{
			name:     "block at end",
			input:    "header\n# sdd:begin path\nexport PATH=...\n# sdd:end path\n",
			expected: "header\n",
		},
		{
			name:     "no block",
			input:    "header\nfooter\n",
			expected: "header\nfooter\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeMarkedBlock(tt.input, "# sdd:begin path", "# sdd:end path")
			if got != tt.expected {
				t.Errorf("removeMarkedBlock() = %q, want %q", got, tt.expected)
			}
		})
	}
}
