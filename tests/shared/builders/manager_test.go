//go:build fast
// +build fast

package builders

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildManager_GetBinary(t *testing.T) {
	t.Parallel()
	bm := NewBuildManager()
	defer bm.Cleanup()

	binaryPath, err := bm.GetBinary("vn")
	if err != nil {
		t.Fatalf("Failed to get binary: %v", err)
	}

	if binaryPath == "" {
		t.Fatal("Binary path is empty")
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("Binary does not exist at path: %s", binaryPath)
	}

	binaryPath2, err := bm.GetBinary("vn")
	if err != nil {
		t.Fatalf("Failed to get cached binary: %v", err)
	}

	if binaryPath != binaryPath2 {
		t.Fatalf("Expected cached binary path %s, got %s", binaryPath, binaryPath2)
	}
}

func TestBuildManager_BuildOnce(t *testing.T) {
	t.Parallel()
	bm := NewBuildManager()
	defer bm.Cleanup()

	binaryPath, err := bm.BuildOnce("test-server", "../../../test-server")
	if err != nil {
		t.Fatalf("Failed to build from source: %v", err)
	}

	if binaryPath == "" {
		t.Fatal("Binary path is empty")
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("Binary does not exist at path: %s", binaryPath)
	}
}

func TestBuildManager_Cleanup(t *testing.T) {
	t.Parallel()
	bm := NewBuildManager()

	binaryPath, err := bm.GetBinary("vn")
	if err != nil {
		t.Fatalf("Failed to get binary: %v", err)
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("Binary does not exist before cleanup: %s", binaryPath)
	}

	bm.Cleanup()

	if _, err := os.Stat(binaryPath); !os.IsNotExist(err) {
		t.Fatalf("Binary still exists after cleanup: %s", binaryPath)
	}
}

func TestBuildManager_SourceHash(t *testing.T) {
	t.Parallel()
	bm := &buildManager{
		binaries: make(map[string]string),
		tempDir:  filepath.Join(os.TempDir(), "test-builds"),
	}

	hash1, err := bm.getSourceHash()
	if err != nil {
		t.Fatalf("Failed to get source hash: %v", err)
	}

	if len(hash1) != 8 {
		t.Fatalf("Expected hash length 8, got %d", len(hash1))
	}

	hash2, err := bm.getSourceHash()
	if err != nil {
		t.Fatalf("Failed to get source hash second time: %v", err)
	}

	if hash1 != hash2 {
		t.Fatalf("Hash inconsistent: %s != %s", hash1, hash2)
	}
}
