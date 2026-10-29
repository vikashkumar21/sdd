package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsTerminal(t *testing.T) {
	// Regular file must not be a terminal
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "temp.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	if isTerminal(f) {
		t.Error("regular file should not be reported as a terminal")
	}

	// Pipe must not be a terminal
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()

	if isTerminal(r) {
		t.Error("pipe reader should not be reported as a terminal")
	}
	if isTerminal(w) {
		t.Error("pipe writer should not be reported as a terminal")
	}

	// Closed file should return false (stat error)
	closedFile, err := os.Create(filepath.Join(dir, "closed.txt"))
	if err != nil {
		t.Fatal(err)
	}
	_ = closedFile.Close()
	_ = os.Remove(closedFile.Name())
	if isTerminal(closedFile) {
		t.Error("deleted/closed file should return false")
	}
}

func TestDefaultIO(t *testing.T) {
	env := defaultIO()
	if env.In == nil || env.Out == nil || env.Err == nil {
		t.Fatal("defaultIO returned nil stream")
	}
	if env.Now == nil {
		t.Fatal("defaultIO Now function is nil")
	}
	if env.Getenv == nil {
		t.Fatal("defaultIO Getenv function is nil")
	}
	if env.Home == nil {
		t.Fatal("defaultIO Home function is nil")
	}
	home := env.Home()
	if home == "" {
		t.Log("HomeDir returned empty string (may happen in isolated environments)")
	}
}

func TestMainExecution(t *testing.T) {
	if os.Getenv("BE_SDD_MAIN") == "1" {
		os.Args = []string{"sdd", "version"}
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestMainExecution")
	cmd.Env = append(os.Environ(), "BE_SDD_MAIN=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("main subprocess exited with error: %v, output: %s", err, out)
	}
	if !strings.Contains(string(out), "sdd ") {
		t.Errorf("expected version in output, got: %s", out)
	}
}
