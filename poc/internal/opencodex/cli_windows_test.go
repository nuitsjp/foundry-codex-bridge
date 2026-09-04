//go:build windows

package opencodex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagedCommandBypassesCommandShim(t *testing.T) {
	pathDir := t.TempDir()
	nodePath := filepath.Join(pathDir, "node.exe")
	ocxShim := filepath.Join(pathDir, "ocx.cmd")
	for _, path := range []string{nodePath, ocxShim} {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", pathDir)

	dataDir := t.TempDir()
	entry := filepath.Join(dataDir, "opencodex", "node_modules", "@bitkyc08", "opencodex", "bin", "ocx.mjs")
	if err := os.MkdirAll(filepath.Dir(entry), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	spec, err := NewLocalRunner(dataDir).command()
	if err != nil {
		t.Fatal(err)
	}
	if spec.path != nodePath {
		t.Fatalf("command path = %q, want %q", spec.path, nodePath)
	}
	if len(spec.prefix) != 1 || spec.prefix[0] != entry {
		t.Fatalf("command prefix = %v, want [%q]", spec.prefix, entry)
	}
}

func TestGlobalCommandShimResolvesToPackageEntry(t *testing.T) {
	pathDir := t.TempDir()
	nodePath := filepath.Join(pathDir, "node.exe")
	ocxShim := filepath.Join(pathDir, "ocx.cmd")
	entry := filepath.Join(pathDir, "node_modules", "@bitkyc08", "opencodex", "bin", "ocx.mjs")
	if err := os.MkdirAll(filepath.Dir(entry), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{nodePath, ocxShim, entry} {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", pathDir)

	spec, err := NewLocalRunner(t.TempDir()).command()
	if err != nil {
		t.Fatal(err)
	}
	if spec.path != nodePath || len(spec.prefix) != 1 || spec.prefix[0] != entry {
		t.Fatalf("command = %q %v, want %q [%q]", spec.path, spec.prefix, nodePath, entry)
	}
}
