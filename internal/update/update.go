// Package update implements native binary self-updating for sdd.
package update

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/vikashkumar21/sdd/internal/version"
)

// DefaultRepo is the default GitHub repository owner/name.
const DefaultRepo = "vikashkumar21/sdd"

// Release represents a GitHub release object.
type Release struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	HTMLURL     string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

// Asset represents a downloadable release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// CheckResult represents the outcome of an update check.
type CheckResult struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	TargetTag       string `json:"target_tag"`
	UpdateAvailable bool   `json:"update_available"`
	AssetName       string `json:"asset_name,omitempty"`
	AssetURL        string `json:"asset_url,omitempty"`
	ChecksumURL     string `json:"checksum_url,omitempty"`
}

// Options configures update discovery, download, and verification.
type Options struct {
	BaseURL        string       // custom API base URL (defaults to GitHub Releases API)
	Client         *http.Client // HTTP client to use
	CurrentVersion string       // current binary version (defaults to version.Version)
	TargetVersion  string       // specific version tag requested (--version)
	GOOS           string       // defaults to runtime.GOOS
	GOARCH         string       // defaults to runtime.GOARCH
	Force          bool         // allow reinstalling matching version
	CheckOnly      bool         // query release only
	DryRun         bool         // verify download without replacing binary
}

// AssetName returns the expected binary asset name for the given OS and architecture.
func AssetName(goos, goarch string) string {
	name := fmt.Sprintf("sdd_%s_%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// ValidateVersionTag validates that a target version tag contains only allowed characters
// (alphanumeric, dot, dash, underscore, plus) and contains no path traversal sequences.
func ValidateVersionTag(tag string) error {
	t := strings.TrimSpace(tag)
	if t == "" {
		return errors.New("version tag cannot be empty")
	}
	if len(t) > 64 {
		return fmt.Errorf("version tag exceeds 64 characters: %s", tag)
	}
	if strings.Contains(t, "..") || strings.Contains(t, "/") || strings.Contains(t, "\\") {
		return fmt.Errorf("invalid version tag %q: path traversal characters not permitted", tag)
	}
	for i := 0; i < len(t); i++ {
		c := t[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_' || c == '+' {
			continue
		}
		return fmt.Errorf("invalid character %q in version tag %q", c, tag)
	}
	return nil
}

type semver struct {
	major, minor, patch int
	prerelease          string
	isParsed            bool
}

func parseSemver(v string) semver {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")

	var sv semver
	if s == "" || s == "dev" {
		return sv
	}

	var pre string
	if idx := strings.IndexByte(s, '-'); idx != -1 {
		pre = s[idx+1:]
		s = s[:idx]
	}
	if idx := strings.IndexByte(s, '+'); idx != -1 {
		s = s[:idx]
	}
	if idx := strings.IndexByte(pre, '+'); idx != -1 {
		pre = pre[:idx]
	}

	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return sv
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return sv
	}
	minor := 0
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return sv
		}
	}
	patch := 0
	if len(parts) > 2 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return sv
		}
	}

	sv.major = major
	sv.minor = minor
	sv.patch = patch
	sv.prerelease = pre
	sv.isParsed = true
	return sv
}

// CompareSemver compares two semantic version strings (with or without 'v' prefix).
// Prerelease identifiers (e.g. -alpha, -beta, -rc1) are compared lexically for lightweight
// dependency-free comparison without external dependencies.
// Returns -1 if v1 < v2, 0 if v1 == v2, and 1 if v1 > v2.
func CompareSemver(v1, v2 string) int {
	if v1 == v2 {
		return 0
	}
	s1 := parseSemver(v1)
	s2 := parseSemver(v2)

	if !s1.isParsed && !s2.isParsed {
		return strings.Compare(v1, v2)
	}
	if !s1.isParsed {
		return -1
	}
	if !s2.isParsed {
		return 1
	}

	if s1.major != s2.major {
		if s1.major < s2.major {
			return -1
		}
		return 1
	}
	if s1.minor != s2.minor {
		if s1.minor < s2.minor {
			return -1
		}
		return 1
	}
	if s1.patch != s2.patch {
		if s1.patch < s2.patch {
			return -1
		}
		return 1
	}

	// Release without pre-release > pre-release
	if s1.prerelease == "" && s2.prerelease != "" {
		return 1
	}
	if s1.prerelease != "" && s2.prerelease == "" {
		return -1
	}
	return strings.Compare(s1.prerelease, s2.prerelease)
}

// Check queries GitHub Releases and returns whether an update is available.
func Check(ctx context.Context, opts Options) (CheckResult, error) {
	var res CheckResult

	goos := opts.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := opts.GOARCH
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	currentVer := opts.CurrentVersion
	if currentVer == "" {
		currentVer = version.Version
	}
	client := opts.Client
	if client == nil {
		client = http.DefaultClient
	}
	baseURL := strings.TrimRight(opts.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.github.com/repos/" + DefaultRepo
	}

	var reqURL string
	if opts.TargetVersion != "" {
		if err := ValidateVersionTag(opts.TargetVersion); err != nil {
			return res, err
		}
		tag := opts.TargetVersion
		if !strings.HasPrefix(tag, "v") && !strings.HasPrefix(tag, "V") {
			tag = "v" + tag
		}
		reqURL = fmt.Sprintf("%s/releases/tags/%s", baseURL, url.PathEscape(tag))
	} else {
		reqURL = fmt.Sprintf("%s/releases/latest", baseURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return res, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("sdd/%s", currentVer))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return res, fmt.Errorf("fetching release info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		if opts.TargetVersion != "" {
			return res, fmt.Errorf("release not found for version %s", opts.TargetVersion)
		}
		return res, errors.New("no releases found")
	}
	if resp.StatusCode != http.StatusOK {
		return res, fmt.Errorf("GitHub API error: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	var rel Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		return res, fmt.Errorf("decoding release response: %w", err)
	}

	assetName := AssetName(goos, goarch)
	var binaryAsset *Asset
	var checksumAsset *Asset

	for i := range rel.Assets {
		a := &rel.Assets[i]
		switch a.Name {
		case assetName:
			binaryAsset = a
		case "SHA256SUMS":
			checksumAsset = a
		}
	}

	if binaryAsset == nil {
		return res, fmt.Errorf("release %s has no binary asset for %s/%s (%s)", rel.TagName, goos, goarch, assetName)
	}
	if checksumAsset == nil {
		return res, fmt.Errorf("release %s has no SHA256SUMS checksum asset", rel.TagName)
	}

	targetVer := strings.TrimPrefix(strings.TrimPrefix(rel.TagName, "v"), "V")
	cmp := CompareSemver(currentVer, targetVer)

	var updateAvailable bool
	if opts.Force {
		updateAvailable = true
	} else if opts.TargetVersion != "" {
		updateAvailable = (cmp != 0)
	} else {
		updateAvailable = (cmp < 0)
	}

	res = CheckResult{
		CurrentVersion:  currentVer,
		LatestVersion:   targetVer,
		TargetTag:       rel.TagName,
		UpdateAvailable: updateAvailable,
		AssetName:       assetName,
		AssetURL:        binaryAsset.BrowserDownloadURL,
		ChecksumURL:     checksumAsset.BrowserDownloadURL,
	}
	return res, nil
}

// ParseChecksum parses SHA256SUMS content looking for the entry corresponding to assetName.
func ParseChecksum(content, assetName string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		hash := strings.ToLower(fields[0])
		filename := strings.TrimPrefix(fields[1], "*")
		filename = filepath.Base(filename)
		if filename == assetName {
			if len(hash) != 64 {
				return "", fmt.Errorf("invalid checksum length for %s: %s", assetName, hash)
			}
			if _, err := hex.DecodeString(hash); err != nil {
				return "", fmt.Errorf("invalid checksum hex for %s: %w", assetName, err)
			}
			return hash, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading checksums: %w", err)
	}
	return "", fmt.Errorf("checksum for %s not found in SHA256SUMS", assetName)
}

func readBounded(r io.Reader, maxBytes int64, resourceName string) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", resourceName, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%s exceeded maximum allowed size of %d bytes", resourceName, maxBytes)
	}
	return data, nil
}

func validateURLScheme(rawURL string, allowHTTP bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid download URL %q: %w", rawURL, err)
	}
	if u.Scheme == "https" {
		return nil
	}
	if allowHTTP && u.Scheme == "http" {
		return nil
	}
	return fmt.Errorf("insecure download scheme %q for %s: HTTPS required", u.Scheme, rawURL)
}

// DownloadAndVerify downloads the release checksums and binary asset, verifying SHA256 integrity.
func DownloadAndVerify(ctx context.Context, res CheckResult, opts Options) ([]byte, error) {
	client := opts.Client
	if client == nil {
		client = http.DefaultClient
	}
	currentVer := opts.CurrentVersion
	if currentVer == "" {
		currentVer = version.Version
	}

	allowHTTP := (opts.BaseURL != "")
	if err := validateURLScheme(res.ChecksumURL, allowHTTP); err != nil {
		return nil, err
	}
	if err := validateURLScheme(res.AssetURL, allowHTTP); err != nil {
		return nil, err
	}

	// 1. Download SHA256SUMS
	chkReq, err := http.NewRequestWithContext(ctx, http.MethodGet, res.ChecksumURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating checksum request: %w", err)
	}
	chkReq.Header.Set("User-Agent", fmt.Sprintf("sdd/%s", currentVer))

	chkResp, err := client.Do(chkReq)
	if err != nil {
		return nil, fmt.Errorf("downloading checksums: %w", err)
	}
	defer func() { _ = chkResp.Body.Close() }()

	if chkResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading checksums: HTTP %d %s", chkResp.StatusCode, chkResp.Status)
	}

	chkData, err := readBounded(chkResp.Body, 1<<20, "SHA256SUMS") // 1MB limit
	if err != nil {
		return nil, err
	}

	expectedHash, err := ParseChecksum(string(chkData), res.AssetName)
	if err != nil {
		return nil, err
	}

	// 2. Download Binary Asset
	binReq, err := http.NewRequestWithContext(ctx, http.MethodGet, res.AssetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating binary download request: %w", err)
	}
	binReq.Header.Set("User-Agent", fmt.Sprintf("sdd/%s", currentVer))

	binResp, err := client.Do(binReq)
	if err != nil {
		return nil, fmt.Errorf("downloading binary asset: %w", err)
	}
	defer func() { _ = binResp.Body.Close() }()

	if binResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading binary asset: HTTP %d %s", binResp.StatusCode, binResp.Status)
	}

	binData, err := readBounded(binResp.Body, 100<<20, "binary asset") // 100MB limit
	if err != nil {
		return nil, err
	}
	if len(binData) == 0 {
		return nil, errors.New("downloaded binary asset is empty")
	}

	// 3. Verify SHA256 Checksum in constant time
	hasher := sha256.New()
	hasher.Write(binData)
	computedHash := hex.EncodeToString(hasher.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(strings.ToLower(computedHash)), []byte(strings.ToLower(expectedHash))) != 1 {
		return nil, fmt.Errorf("checksum mismatch for %s: expected %s, got %s", res.AssetName, expectedHash, computedHash)
	}

	return binData, nil
}

// ReplaceExecutable safely replaces the binary at execPath with newBytes.
// If execPath is empty, os.Executable() is used.
func ReplaceExecutable(execPath string, newBytes []byte) error {
	if execPath == "" {
		p, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolving current executable path: %w", err)
		}
		execPath = p
	}

	absPath, err := filepath.Abs(execPath)
	if err == nil {
		execPath = absPath
	}

	// Resolve symlinks so the real binary is replaced instead of destroying symlink
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		realPath = execPath
	}
	realPath = filepath.Clean(realPath)

	fi, err := os.Stat(realPath)
	if err == nil && fi.IsDir() {
		return fmt.Errorf("target executable path is a directory: %s", realPath)
	}

	dir := filepath.Dir(realPath)
	tmpFile, err := os.CreateTemp(dir, "sdd-update-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary staging file: %w", err)
	}
	tmpPath := tmpFile.Name()

	cleanup := func() {
		_ = os.Remove(tmpPath)
	}

	if _, err := tmpFile.Write(newBytes); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return fmt.Errorf("writing staged binary: %w", err)
	}

	if err := tmpFile.Chmod(0755); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return fmt.Errorf("setting permissions on staged binary: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		cleanup()
		return fmt.Errorf("closing staged binary: %w", err)
	}

	return swapExecutable(tmpPath, realPath)
}

func swapExecutable(tmpPath, realPath string) error {
	return swapExecutableOS(tmpPath, realPath, runtime.GOOS)
}

func swapExecutableOS(tmpPath, realPath, goos string) error {
	if goos == "windows" {
		oldPath := realPath + ".old"
		_ = os.Remove(oldPath) // remove stale previous backup if present

		if err := os.Rename(realPath, oldPath); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("backing up running executable: %w", err)
		}

		if err := os.Rename(tmpPath, realPath); err != nil {
			// Rollback to original executable
			_ = os.Rename(oldPath, realPath)
			_ = os.Remove(tmpPath)
			return fmt.Errorf("swapping executable: %w (original restored)", err)
		}

		_ = os.Remove(oldPath)
		return nil
	}

	// Unix atomic rename
	if err := os.Rename(tmpPath, realPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("swapping executable: %w", err)
	}
	return nil
}
