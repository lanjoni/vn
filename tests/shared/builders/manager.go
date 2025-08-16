package builders

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

const (
	tempDirMode = 0755
)

type BuildManager interface {
	GetBinary(name string) (string, error)
	BuildOnce(name, source string) (string, error)
	Cleanup()
}

type buildManager struct {
	mu       sync.Mutex
	binaries map[string]string
	tempDir  string
}

func NewBuildManager() BuildManager {
	tempDir := filepath.Join(os.TempDir(), "vn-test-builds")
	//nolint:errcheck
	_ = os.MkdirAll(tempDir, tempDirMode) // Best effort, ignore error

	return &buildManager{
		binaries: make(map[string]string),
		tempDir:  tempDir,
	}
}

func (bm *buildManager) GetBinary(name string) (string, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if path, exists := bm.binaries[name]; exists {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		delete(bm.binaries, name)
	}

	return bm.buildBinary(name)
}

func (bm *buildManager) BuildOnce(name, source string) (string, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if path, exists := bm.binaries[name]; exists {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		delete(bm.binaries, name)
	}

	return bm.buildBinaryFromSource(name, source)
}

func (bm *buildManager) buildBinary(name string) (string, error) {
	hash, err := bm.getSourceHash()
	if err != nil {
		return "", fmt.Errorf("failed to get source hash: %w", err)
	}

	binaryPath := filepath.Join(bm.tempDir, fmt.Sprintf("%s-%s", name, hash))

	if _, err := os.Stat(binaryPath); err == nil {
		bm.binaries[name] = binaryPath
		return binaryPath, nil
	}

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to build binary %s: %w", name, err)
	}

	bm.binaries[name] = binaryPath
	return binaryPath, nil
}

func (bm *buildManager) buildBinaryFromSource(name, source string) (string, error) {
	hash, err := bm.getSourceHashFromPath(source)
	if err != nil {
		return "", fmt.Errorf("failed to get source hash: %w", err)
	}

	binaryPath := filepath.Join(bm.tempDir, fmt.Sprintf("%s-%s", name, hash))

	if _, err := os.Stat(binaryPath); err == nil {
		bm.binaries[name] = binaryPath
		return binaryPath, nil
	}

	cmd := exec.Command("go", "build", "-o", binaryPath, source)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to build binary %s from %s: %w", name, source, err)
	}

	bm.binaries[name] = binaryPath
	return binaryPath, nil
}

func (bm *buildManager) getSourceHash() (string, error) {
	return bm.getSourceHashFromPath(".")
}

func (bm *buildManager) getSourceHashFromPath(path string) (string, error) {
	hasher := sha256.New()

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(filePath) != ".go" {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(hasher, file); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))[:8], nil
}

func (bm *buildManager) Cleanup() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for _, path := range bm.binaries {
		os.Remove(path)
	}

	bm.binaries = make(map[string]string)
	os.RemoveAll(bm.tempDir)
}
