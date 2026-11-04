package install

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallScriptPOSIX(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "scripts", "install.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("install.sh not found: %v", err)
	}

	cmd := exec.Command("sh", "-n", scriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("POSIX syntax check failed: %v\nOutput: %s", err, string(output))
	}
}

func TestInstallScriptEndToEnd(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("skipping shell script test on non-Unix platform")
	}

	arch := runtime.GOARCH
	goos := runtime.GOOS
	assetName := fmt.Sprintf("sdd_%s_%s", goos, arch)

	serveDir := t.TempDir()
	mockBinaryContent := []byte("#!/bin/sh\nif [ \"$1\" = \"install\" ]; then echo \"Mock installer executed\"; exit 0; fi\nexit 1\n")
	assetPath := filepath.Join(serveDir, assetName)
	if err := os.WriteFile(assetPath, mockBinaryContent, 0o755); err != nil {
		t.Fatal(err)
	}

	hash := sha256.Sum256(mockBinaryContent)
	hashHex := fmt.Sprintf("%x", hash)
	sumsContent := fmt.Sprintf("%s  dist/%s\n", hashHex, assetName)
	sumsPath := filepath.Join(serveDir, "SHA256SUMS")
	if err := os.WriteFile(sumsPath, []byte(sumsContent), 0o644); err != nil {
		t.Fatal(err)
	}

	scriptPath, err := filepath.Abs(filepath.Join("..", "..", "scripts", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}

	baseURL := "file://" + filepath.ToSlash(serveDir)

	t.Run("successful install and checksum verification", func(t *testing.T) {
		cmd := exec.Command("sh", scriptPath)
		cmd.Env = append(os.Environ(),
			"SDD_BASE_URL="+baseURL,
			"SDD_DRY_RUN=1",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("install.sh failed: %v\nOutput:\n%s", err, string(out))
		}

		s := string(out)
		if !strings.Contains(s, "Downloading sdd") {
			t.Errorf("expected 'Downloading sdd' in output: %s", s)
		}
		if !strings.Contains(s, "Verified SHA256 checksum") {
			t.Errorf("expected checksum verification in output: %s", s)
		}
		if !strings.Contains(s, "Mock installer executed") {
			t.Errorf("expected mock installer execution: %s", s)
		}
	})

	t.Run("checksum mismatch failure", func(t *testing.T) {
		corruptDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(corruptDir, assetName), mockBinaryContent, 0o755); err != nil {
			t.Fatal(err)
		}
		badSums := "badhash1234567890badhash1234567890badhash1234567890badhash1234567890  dist/" + assetName + "\n"
		if err := os.WriteFile(filepath.Join(corruptDir, "SHA256SUMS"), []byte(badSums), 0o644); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command("sh", scriptPath)
		cmd.Env = append(os.Environ(),
			"SDD_BASE_URL=file://"+filepath.ToSlash(corruptDir),
			"SDD_DRY_RUN=1",
		)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected install.sh to fail on corrupt checksum, but got success\nOutput:\n%s", string(out))
		}

		s := string(out)
		if !strings.Contains(s, "SHA256 checksum mismatch") {
			t.Errorf("expected checksum mismatch error, got: %s", s)
		}
	})
}
