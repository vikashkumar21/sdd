package coordinator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadModulePath(t *testing.T) {
	root := t.TempDir()
	goModContent := "module example.com/my/project\n\ngo 1.27\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goModContent), 0o644); err != nil {
		t.Fatal(err)
	}

	mod, err := ReadModulePath(root)
	if err != nil {
		t.Fatalf("ReadModulePath failed: %v", err)
	}
	if mod != "example.com/my/project" {
		t.Errorf("expected example.com/my/project, got %q", mod)
	}
}

func TestBlastRadiusComputation(t *testing.T) {
	root := t.TempDir()

	// Setup mock module structure:
	// module: test.mod
	// pkg/a -> imports nothing
	// pkg/b -> imports test.mod/pkg/a
	// pkg/c -> imports test.mod/pkg/b (transitive import of a)
	// pkg/d -> imports nothing (independent)
	goMod := "module test.mod\n\ngo 1.27\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{"pkg/a", "pkg/b", "pkg/c", "pkg/d"} {
		if err := os.MkdirAll(filepath.Join(root, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	fileA := "package a\n"
	fileB := "package b\nimport _ \"test.mod/pkg/a\"\n"
	fileC := "package c\nimport _ \"test.mod/pkg/b\"\n"
	fileD := "package d\n"

	_ = os.WriteFile(filepath.Join(root, "pkg/a/a.go"), []byte(fileA), 0o644)
	_ = os.WriteFile(filepath.Join(root, "pkg/b/b.go"), []byte(fileB), 0o644)
	_ = os.WriteFile(filepath.Join(root, "pkg/c/c.go"), []byte(fileC), 0o644)
	_ = os.WriteFile(filepath.Join(root, "pkg/d/d.go"), []byte(fileD), 0o644)

	// 1. Modifying pkg/a should cascade to pkg/b and pkg/c, but NOT pkg/d
	res, err := ComputeBlastRadius(root, []string{"pkg/a/a.go"})
	if err != nil {
		t.Fatalf("ComputeBlastRadius failed: %v", err)
	}

	if len(res.ModifiedPackages) != 1 || res.ModifiedPackages[0] != "pkg/a" {
		t.Errorf("expected modified pkg/a, got %+v", res.ModifiedPackages)
	}
	if len(res.DownstreamPackages) != 2 {
		t.Fatalf("expected 2 downstream packages (pkg/b, pkg/c), got: %+v", res.DownstreamPackages)
	}
	if res.DownstreamPackages[0] != "pkg/b" || res.DownstreamPackages[1] != "pkg/c" {
		t.Errorf("unexpected downstream packages: %+v", res.DownstreamPackages)
	}
	if len(res.AllAffectedPackages) != 3 {
		t.Fatalf("expected 3 affected packages, got: %+v", res.AllAffectedPackages)
	}

	// 2. Modifying pkg/d should have 0 downstream packages
	resD, err := ComputeBlastRadius(root, []string{"pkg/d/d.go"})
	if err != nil {
		t.Fatalf("ComputeBlastRadius for D failed: %v", err)
	}
	if len(resD.DownstreamPackages) != 0 {
		t.Errorf("expected 0 downstream packages for independent pkg/d, got %+v", resD.DownstreamPackages)
	}
	if len(resD.AllAffectedPackages) != 1 || resD.AllAffectedPackages[0] != "pkg/d" {
		t.Errorf("expected only pkg/d, got %+v", resD.AllAffectedPackages)
	}
}

func TestBlastRadiusOnCurrentRepo(t *testing.T) {
	// Run against current repository to verify real-world behavior
	res, err := ComputeBlastRadius(".", []string{"internal/deconflict/deconflict.go"})
	if err != nil {
		t.Fatalf("ComputeBlastRadius on . failed: %v", err)
	}

	if len(res.ModifiedPackages) != 1 || res.ModifiedPackages[0] != "internal/deconflict" {
		t.Errorf("expected internal/deconflict, got: %+v", res.ModifiedPackages)
	}

	// In current repo, internal/cli, internal/status, internal/check all import internal/deconflict
	foundCLI := false
	foundCheck := false
	foundStatus := false
	for _, p := range res.DownstreamPackages {
		if p == "internal/cli" {
			foundCLI = true
		}
		if p == "internal/check" {
			foundCheck = true
		}
		if p == "internal/status" {
			foundStatus = true
		}
	}
	if !foundCLI || !foundCheck || !foundStatus {
		t.Errorf("expected downstream packages to include cli, check, and status: %+v", res.DownstreamPackages)
	}
}
