package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		v1, v2 string
		want   int
	}{
		{"1.4.0", "1.5.0", -1},
		{"1.5.0", "1.4.0", 1},
		{"1.4.0", "1.4.0", 0},
		{"v1.4.0", "1.4.0", 0},
		{"V1.4.0", "v1.4.0", 0},
		{"1.4.0", "1.4.1", -1},
		{"1.4.1", "1.4.0", 1},
		{"1.4.0-rc1", "1.4.0", -1},
		{"1.4.0", "1.4.0-rc1", 1},
		{"1.4.0-alpha", "1.4.0-beta", -1},
		{"1.4.0-beta", "1.4.0-alpha", 1},
		{"dev", "1.4.0", -1},
		{"1.4.0", "dev", 1},
		{"dev", "dev", 0},
		{"1.4", "1.4.0", 0},
		{"2.0.0", "1.9.9", 1},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.v1, tt.v2), func(t *testing.T) {
			got := CompareSemver(tt.v1, tt.v2)
			if got != tt.want {
				t.Errorf("CompareSemver(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestAssetName(t *testing.T) {
	if got := AssetName("darwin", "arm64"); got != "sdd_darwin_arm64" {
		t.Errorf("got %q, want sdd_darwin_arm64", got)
	}
	if got := AssetName("linux", "amd64"); got != "sdd_linux_amd64" {
		t.Errorf("got %q, want sdd_linux_amd64", got)
	}
	if got := AssetName("windows", "amd64"); got != "sdd_windows_amd64.exe" {
		t.Errorf("got %q, want sdd_windows_amd64.exe", got)
	}
}

func TestParseChecksum(t *testing.T) {
	sample := `# SHA256 checksums
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  sdd_darwin_arm64
ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad *sdd_windows_amd64.exe
`

	t.Run("valid unix entry", func(t *testing.T) {
		hash, err := ParseChecksum(sample, "sdd_darwin_arm64")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hash != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
			t.Errorf("unexpected hash: %s", hash)
		}
	})

	t.Run("valid windows entry with asterisk", func(t *testing.T) {
		hash, err := ParseChecksum(sample, "sdd_windows_amd64.exe")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hash != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
			t.Errorf("unexpected hash: %s", hash)
		}
	})

	t.Run("missing asset", func(t *testing.T) {
		_, err := ParseChecksum(sample, "sdd_linux_amd64")
		if err == nil {
			t.Fatal("expected error for missing asset, got nil")
		}
	})

	t.Run("invalid hash length", func(t *testing.T) {
		bad := "short123  sdd_linux_amd64\n"
		_, err := ParseChecksum(bad, "sdd_linux_amd64")
		if err == nil {
			t.Fatal("expected error for short hash, got nil")
		}
	})

	t.Run("invalid hash hex characters", func(t *testing.T) {
		bad := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz  sdd_linux_amd64\n"
		_, err := ParseChecksum(bad, "sdd_linux_amd64")
		if err == nil {
			t.Fatal("expected error for non-hex hash, got nil")
		}
	})
}

func TestCheck(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho sdd 1.5.0\n")
	hasher := sha256.New()
	hasher.Write(binaryContent)
	binHash := hex.EncodeToString(hasher.Sum(nil))

	checksumsContent := fmt.Sprintf("%s  sdd_darwin_arm64\n", binHash)

	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		if ua := req.Header.Get("User-Agent"); ua == "" {
			t.Errorf("expected User-Agent header, got empty")
		}

		path := req.URL.Path
		switch path {
		case "/repos/vikashkumar21/sdd/releases/latest":
			rel := Release{
				TagName:     "v1.5.0",
				Name:        "Release 1.5.0",
				PublishedAt: "2026-11-10T12:00:00Z",
				Assets: []Asset{
					{
						Name:               "sdd_darwin_arm64",
						BrowserDownloadURL: "https://mock.local/download/sdd_darwin_arm64",
						Size:               int64(len(binaryContent)),
					},
					{
						Name:               "SHA256SUMS",
						BrowserDownloadURL: "https://mock.local/download/SHA256SUMS",
						Size:               int64(len(checksumsContent)),
					},
				},
			}
			data, _ := json.Marshal(rel)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(data)),
				Header:     make(http.Header),
			}, nil
		case "/repos/vikashkumar21/sdd/releases/tags/v1.3.0":
			rel := Release{
				TagName:     "v1.3.0",
				Name:        "Release 1.3.0",
				PublishedAt: "2026-11-01T12:00:00Z",
				Assets: []Asset{
					{
						Name:               "sdd_darwin_arm64",
						BrowserDownloadURL: "https://mock.local/download/sdd_darwin_arm64_v1.3.0",
						Size:               int64(len(binaryContent)),
					},
					{
						Name:               "SHA256SUMS",
						BrowserDownloadURL: "https://mock.local/download/SHA256SUMS_v1.3.0",
						Size:               int64(len(checksumsContent)),
					},
				},
			}
			data, _ := json.Marshal(rel)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(data)),
				Header:     make(http.Header),
			}, nil
		case "/repos/vikashkumar21/sdd/releases/tags/v9.9.9":
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"message":"Not Found"}`))),
				Header:     make(http.Header),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(bytes.NewReader([]byte("not found"))),
				Header:     make(http.Header),
			}, nil
		}
	})

	ctx := context.Background()

	t.Run("newer version available", func(t *testing.T) {
		opts := Options{
			BaseURL:        "https://mock.local/repos/vikashkumar21/sdd",
			Client:         client,
			CurrentVersion: "1.4.0",
			GOOS:           "darwin",
			GOARCH:         "arm64",
		}
		res, err := Check(ctx, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.UpdateAvailable {
			t.Errorf("expected UpdateAvailable to be true")
		}
		if res.LatestVersion != "1.5.0" {
			t.Errorf("got LatestVersion %q, want 1.5.0", res.LatestVersion)
		}
		if res.AssetName != "sdd_darwin_arm64" {
			t.Errorf("got AssetName %q, want sdd_darwin_arm64", res.AssetName)
		}
	})

	t.Run("already up to date", func(t *testing.T) {
		opts := Options{
			BaseURL:        "https://mock.local/repos/vikashkumar21/sdd",
			Client:         client,
			CurrentVersion: "1.5.0",
			GOOS:           "darwin",
			GOARCH:         "arm64",
		}
		res, err := Check(ctx, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.UpdateAvailable {
			t.Errorf("expected UpdateAvailable to be false")
		}
	})

	t.Run("already up to date with force", func(t *testing.T) {
		opts := Options{
			BaseURL:        "https://mock.local/repos/vikashkumar21/sdd",
			Client:         client,
			CurrentVersion: "1.5.0",
			Force:          true,
			GOOS:           "darwin",
			GOARCH:         "arm64",
		}
		res, err := Check(ctx, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.UpdateAvailable {
			t.Errorf("expected UpdateAvailable to be true with Force")
		}
	})

	t.Run("target specific version", func(t *testing.T) {
		opts := Options{
			BaseURL:        "https://mock.local/repos/vikashkumar21/sdd",
			Client:         client,
			CurrentVersion: "1.4.0",
			TargetVersion:  "1.3.0",
			GOOS:           "darwin",
			GOARCH:         "arm64",
		}
		res, err := Check(ctx, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.UpdateAvailable {
			t.Errorf("expected UpdateAvailable to be true when targeting different version")
		}
		if res.LatestVersion != "1.3.0" {
			t.Errorf("got %q, want 1.3.0", res.LatestVersion)
		}
	})

	t.Run("release not found", func(t *testing.T) {
		opts := Options{
			BaseURL:        "https://mock.local/repos/vikashkumar21/sdd",
			Client:         client,
			CurrentVersion: "1.4.0",
			TargetVersion:  "9.9.9",
			GOOS:           "darwin",
			GOARCH:         "arm64",
		}
		_, err := Check(ctx, opts)
		if err == nil {
			t.Fatal("expected error for non-existent release, got nil")
		}
	})
}

func TestDownloadAndVerify(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho sdd updated\n")
	hasher := sha256.New()
	hasher.Write(binaryContent)
	binHash := hex.EncodeToString(hasher.Sum(nil))

	checksumsContent := fmt.Sprintf("%s  sdd_darwin_arm64\n", binHash)

	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/download/sdd_darwin_arm64":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(binaryContent)),
				Header:     make(http.Header),
			}, nil
		case "/download/corrupt_binary":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("corrupt-bytes"))),
				Header:     make(http.Header),
			}, nil
		case "/download/SHA256SUMS":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(checksumsContent))),
				Header:     make(http.Header),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(bytes.NewReader([]byte("not found"))),
				Header:     make(http.Header),
			}, nil
		}
	})

	ctx := context.Background()

	t.Run("successful download and verification", func(t *testing.T) {
		res := CheckResult{
			AssetName:   "sdd_darwin_arm64",
			AssetURL:    "https://mock.local/download/sdd_darwin_arm64",
			ChecksumURL: "https://mock.local/download/SHA256SUMS",
		}
		opts := Options{
			Client:         client,
			CurrentVersion: "1.4.0",
		}

		bytes, err := DownloadAndVerify(ctx, res, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(bytes) != string(binaryContent) {
			t.Errorf("downloaded bytes mismatch: got %q, want %q", string(bytes), string(binaryContent))
		}
	})

	t.Run("checksum mismatch detection", func(t *testing.T) {
		res := CheckResult{
			AssetName:   "sdd_darwin_arm64",
			AssetURL:    "https://mock.local/download/corrupt_binary",
			ChecksumURL: "https://mock.local/download/SHA256SUMS",
		}
		opts := Options{
			Client:         client,
			CurrentVersion: "1.4.0",
		}

		_, err := DownloadAndVerify(ctx, res, opts)
		if err == nil {
			t.Fatal("expected checksum mismatch error, got nil")
		}
	})
}

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	originalPath := filepath.Join(dir, "sdd")
	if err := os.WriteFile(originalPath, []byte("version-1.4.0"), 0755); err != nil {
		t.Fatalf("writing original binary: %v", err)
	}

	newBytes := []byte("version-1.5.0")

	t.Run("direct file replacement", func(t *testing.T) {
		if err := ReplaceExecutable(originalPath, newBytes); err != nil {
			t.Fatalf("ReplaceExecutable failed: %v", err)
		}

		data, err := os.ReadFile(originalPath)
		if err != nil {
			t.Fatalf("reading replaced binary: %v", err)
		}
		if string(data) != "version-1.5.0" {
			t.Errorf("got content %q, want version-1.5.0", string(data))
		}
	})

	t.Run("symlink resolution replacement", func(t *testing.T) {
		binDir := t.TempDir()
		realBin := filepath.Join(binDir, "real_sdd")
		if err := os.WriteFile(realBin, []byte("old-real"), 0755); err != nil {
			t.Fatalf("writing real binary: %v", err)
		}

		symlinkPath := filepath.Join(binDir, "sdd_symlink")
		if err := os.Symlink(realBin, symlinkPath); err != nil {
			t.Skipf("symlinks not supported on this platform: %v", err)
		}

		updatedBytes := []byte("new-real")
		if err := ReplaceExecutable(symlinkPath, updatedBytes); err != nil {
			t.Fatalf("ReplaceExecutable failed via symlink: %v", err)
		}

		// Ensure symlink still exists and points to realBin
		fi, err := os.Lstat(symlinkPath)
		if err != nil {
			t.Fatalf("lstat symlink: %v", err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("expected %s to still be a symlink", symlinkPath)
		}

		// Ensure real binary was updated
		data, err := os.ReadFile(realBin)
		if err != nil {
			t.Fatalf("reading real binary: %v", err)
		}
		if string(data) != "new-real" {
			t.Errorf("real binary content = %q, want new-real", string(data))
		}
	})

	t.Run("windows atomic swap with old backup", func(t *testing.T) {
		winDir := t.TempDir()
		winBin := filepath.Join(winDir, "sdd.exe")
		if err := os.WriteFile(winBin, []byte("win-1.4.0"), 0755); err != nil {
			t.Fatalf("writing win binary: %v", err)
		}

		tmpStaged := filepath.Join(winDir, "sdd-tmp.bin")
		if err := os.WriteFile(tmpStaged, []byte("win-1.5.0"), 0755); err != nil {
			t.Fatalf("writing tmp binary: %v", err)
		}

		if err := swapExecutableOS(tmpStaged, winBin, "windows"); err != nil {
			t.Fatalf("swapExecutableOS for windows failed: %v", err)
		}

		data, err := os.ReadFile(winBin)
		if err != nil {
			t.Fatalf("reading replaced win binary: %v", err)
		}
		if string(data) != "win-1.5.0" {
			t.Errorf("got %q, want win-1.5.0", string(data))
		}
	})

	t.Run("target is directory error", func(t *testing.T) {
		targetDir := t.TempDir()
		err := ReplaceExecutable(targetDir, []byte("payload"))
		if err == nil {
			t.Fatal("expected error replacing directory, got nil")
		}
		if !strings.Contains(err.Error(), "target executable path is a directory") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestValidateVersionTag(t *testing.T) {
	valid := []string{"1.5.0", "v1.5.0", "v1.5.0-rc1", "v1.5.0+build123", "2.0.0_beta", "v1-preview"}
	for _, v := range valid {
		if err := ValidateVersionTag(v); err != nil {
			t.Errorf("ValidateVersionTag(%q) unexpected error: %v", v, err)
		}
	}

	invalid := []struct {
		tag  string
		desc string
	}{
		{"", "empty"},
		{"   ", "whitespace only"},
		{"../../evil", "path traversal parent"},
		{"v1.0/tags", "slash"},
		{"v1.0\\tags", "backslash"},
		{"v1.0?foo=bar", "query char"},
		{"v1.0#fragment", "fragment char"},
		{"v1.0;rm -rf", "semicolon"},
		{"v1.0\x00null", "null byte"},
		{strings.Repeat("a", 65), "too long"},
	}

	for _, tt := range invalid {
		t.Run(tt.desc, func(t *testing.T) {
			if err := ValidateVersionTag(tt.tag); err == nil {
				t.Errorf("ValidateVersionTag(%q) expected error, got nil", tt.tag)
			}
		})
	}
}

func TestValidateURLScheme(t *testing.T) {
	t.Run("https scheme valid", func(t *testing.T) {
		if err := validateURLScheme("https://github.com/vikashkumar21/sdd/releases", false); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("http scheme disallowed without allowHTTP", func(t *testing.T) {
		if err := validateURLScheme("http://github.com/vikashkumar21/sdd/releases", false); err == nil {
			t.Errorf("expected error for unencrypted http, got nil")
		}
	})

	t.Run("http scheme allowed with allowHTTP", func(t *testing.T) {
		if err := validateURLScheme("http://mock.local/releases", true); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("file scheme disallowed", func(t *testing.T) {
		if err := validateURLScheme("file:///etc/passwd", true); err == nil {
			t.Errorf("expected error for file scheme, got nil")
		}
	})
}

func TestDownloadAndVerify_BoundedAndEmpty(t *testing.T) {
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/empty_binary":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte{}))}, nil
		case "/oversized_sums":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), (1<<20)+10)))}, nil
		case "/SHA256SUMS":
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader([]byte("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  sdd_darwin_arm64\n")))}, nil
		default:
			return &http.Response{StatusCode: 404, Body: io.NopCloser(bytes.NewReader([]byte("not found")))}, nil
		}
	})

	ctx := context.Background()

	t.Run("empty binary rejected", func(t *testing.T) {
		res := CheckResult{
			AssetName:   "sdd_darwin_arm64",
			AssetURL:    "https://mock.local/empty_binary",
			ChecksumURL: "https://mock.local/SHA256SUMS",
		}
		opts := Options{Client: client, CurrentVersion: "1.4.0", BaseURL: "https://mock.local"}
		_, err := DownloadAndVerify(ctx, res, opts)
		if err == nil {
			t.Fatal("expected error for empty binary, got nil")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("expected empty error, got: %v", err)
		}
	})

	t.Run("oversized checksum rejected", func(t *testing.T) {
		res := CheckResult{
			AssetName:   "sdd_darwin_arm64",
			AssetURL:    "https://mock.local/empty_binary",
			ChecksumURL: "https://mock.local/oversized_sums",
		}
		opts := Options{Client: client, CurrentVersion: "1.4.0", BaseURL: "https://mock.local"}
		_, err := DownloadAndVerify(ctx, res, opts)
		if err == nil {
			t.Fatal("expected error for oversized checksums, got nil")
		}
		if !strings.Contains(err.Error(), "exceeded maximum") {
			t.Errorf("expected size limit error, got: %v", err)
		}
	})
}
