package volume

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"mitl/internal/detector"
)

func TestManager_Mounts_NodeAndPHP(t *testing.T) {
	// Use a temp project with lockfiles to exercise hash logic
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("lockfileVersion: 9"), 0644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"t"}`), 0644)
	os.WriteFile(filepath.Join(dir, "composer.lock"), []byte(`{}`), 0644)

	// Runtime "true" ignores args and exits 0 -> avoids creating real volumes
	vm := NewManager("true", dir)

	node := vm.GetMounts(detector.TypeNodeGeneric)
	if !containsFlag(node, "/app/node_modules") {
		t.Fatalf("expected node_modules mount, got %v", node)
	}
	if !containsFlag(node, "/root/.local/share/pnpm/store") {
		t.Fatalf("expected pnpm store mount, got %v", node)
	}

	php := vm.GetMounts(detector.TypePHPLaravel)
	if !containsFlag(php, "/app/vendor") {
		t.Fatalf("expected vendor mount, got %v", php)
	}
}

func TestManager_InterceptNodeCommand(t *testing.T) {
	vm := NewManager("true", t.TempDir())
	// npm ci -> pnpm install with fallback
	out := vm.InterceptNodeCommand([]string{"npm", "ci"})
	s := strings.Join(out, " ")
	if !strings.Contains(s, "pnpm install") {
		t.Fatalf("expected pnpm install, got %s", s)
	}
	// yarn add -> pnpm add
	out = vm.InterceptNodeCommand([]string{"yarn", "add", "leftpad"})
	s = strings.Join(out, " ")
	if !strings.Contains(s, "pnpm add leftpad") {
		t.Fatalf("expected pnpm add, got %s", s)
	}
	// already pnpm -> unchanged
	out = vm.InterceptNodeCommand([]string{"pnpm", "install"})
	if out[0] != "pnpm" {
		t.Fatalf("expected pnpm passthrough, got %v", out)
	}
	// npm run maps to pnpm run
	out = vm.InterceptNodeCommand([]string{"npm", "run", "test"})
	s = strings.Join(out, " ")
	if !strings.Contains(s, "pnpm run test") {
		t.Fatalf("expected pnpm run, got %s", s)
	}
	// yarn remove maps to pnpm remove
	out = vm.InterceptNodeCommand([]string{"yarn", "remove", "dep"})
	s = strings.Join(out, " ")
	if !strings.Contains(s, "pnpm remove dep") {
		t.Fatalf("expected pnpm remove, got %s", s)
	}
}

func TestManager_LockfileHashing(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	vm := NewManager("true", dir)
	// Ensure consistent name prefix
	name := vm.getOrCreateVolume(VolumeTypePnpmModules)
	if !strings.Contains(name, "pnpm-modules") {
		t.Fatalf("unexpected volume name: %s", name)
	}
}

func containsFlag(flags []string, substr string) bool {
	for _, f := range flags {
		if strings.Contains(f, substr) {
			return true
		}
	}
	return false
}

// smoke test to ensure GetMounts covers platform branches
func TestManager_GetMounts_GoPython(t *testing.T) {
	vm := NewManager("true", t.TempDir())
	_ = vm.getGoMounts()
	_ = vm.getPythonMounts()
	// Just ensure no panic and mounts include expected paths
	g := vm.GetMounts(detector.TypeGoModule)
	p := vm.GetMounts(detector.TypePythonGeneric)
	if runtime.GOOS != "windows" { // simple path check
		if !containsFlag(g, "/root/.cache/go-build") {
			t.Fatalf("expected go-build mount, got %v", g)
		}
		if !containsFlag(p, "/app/.venv") {
			t.Fatalf("expected venv mount, got %v", p)
		}
	}
}

func TestManager_VolumePrimitives(t *testing.T) {
	tmp := t.TempDir()
	// error path for ensurePnpmStore (runtime 'false' always fails)
	vmErr := &Manager{runtime: "false", projectRoot: tmp, metadata: make(map[string]VolumeMetadata), metadataPath: filepath.Join(tmp, "vol.json"), pnpmStore: "store"}
	_ = vmErr.ensurePnpmStore() // may error; just exercise path

	vm := NewManager("true", tmp)
	// volumeExists fallback branch with runtime 'false'
	ok, _ := vmErr.volumeExists("does-not-exist")
	if ok {
		t.Fatalf("expected volumeExists false")
	}
	// create/delete (runtime 'true')
	if err := vm.createVolume("vol1", VolumeTypeVendor, "deadbeef"); err != nil {
		t.Fatalf("createVolume: %v", err)
	}
	if err := vm.deleteVolume("vol1"); err != nil {
		t.Fatalf("deleteVolume: %v", err)
	}
}
