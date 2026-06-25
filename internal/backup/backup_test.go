package backup_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"termius-tools/internal/backup"
)

type fakePlatform struct {
	dataDir string
	key     string
	running bool
}

func (f *fakePlatform) DataDir() (string, error)     { return f.dataDir, nil }
func (f *fakePlatform) GetKey() (string, error)      { return f.key, nil }
func (f *fakePlatform) SetKey(key string) error      { f.key = key; return nil }
func (f *fakePlatform) IsTermiusRunning() bool       { return f.running }
func (f *fakePlatform) CloseHint() string            { return "close termius" }
func (f *fakePlatform) OSName() string               { return "test" }

func createFakeTermiusData(t *testing.T) string {
	t.Helper()
	base := t.TempDir()

	dirs := []string{
		filepath.Join("Local Storage", "leveldb"),
		filepath.Join("IndexedDB", "file__0.indexeddb.leveldb"),
	}
	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(base, dir), 0700)
	}

	files := map[string]string{
		filepath.Join("Local Storage", "leveldb", "CURRENT"):                         "MANIFEST-000001\n",
		filepath.Join("Local Storage", "leveldb", "000001.ldb"):                      "fake-leveldb-data",
		filepath.Join("IndexedDB", "file__0.indexeddb.leveldb", "000001.ldb"):        "fake-idb-data",
		"window-state.json": `{"x":0}`,
		"Preferences":       `{"test":true}`,
	}
	for name, content := range files {
		os.WriteFile(filepath.Join(base, name), []byte(content), 0600)
	}
	return base
}

func TestBackupAndInspect(t *testing.T) {
	dataDir := createFakeTermiusData(t)
	outPath := filepath.Join(t.TempDir(), "test.tbk")

	p := &fakePlatform{dataDir: dataDir, key: "test-key"}
	result, err := backup.Backup(p, outPath, true)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if result.FileCount != 5 {
		t.Errorf("file count: got %d, want 5", result.FileCount)
	}

	info, err := backup.Inspect(outPath)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if info.Metadata.LocalKey != "test-key" {
		t.Errorf("local key: got %q", info.Metadata.LocalKey)
	}
	if len(info.Files) != 5 {
		t.Errorf("inspect files: got %d, want 5", len(info.Files))
	}
}

func TestRestore(t *testing.T) {
	dataDir := createFakeTermiusData(t)
	outPath := filepath.Join(t.TempDir(), "test.tbk")

	src := &fakePlatform{dataDir: dataDir, key: "original-key"}
	backup.Backup(src, outPath, true)

	restoreDir := createFakeTermiusData(t)
	dst := &fakePlatform{dataDir: restoreDir, key: "old-key"}

	result, err := backup.Restore(dst, outPath)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if result.Restored != 5 {
		t.Errorf("restored: got %d, want 5", result.Restored)
	}
	if dst.key != "original-key" {
		t.Errorf("key not restored: got %q", dst.key)
	}
}

func TestBackupFileFormat(t *testing.T) {
	dataDir := createFakeTermiusData(t)
	outPath := filepath.Join(t.TempDir(), "test.tbk")

	p := &fakePlatform{dataDir: dataDir, key: "k"}
	backup.Backup(p, outPath, true)

	data, _ := os.ReadFile(outPath)
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var foundMeta bool
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(tr)
		if header.Name == "metadata.json" {
			foundMeta = true
			var meta backup.Metadata
			json.Unmarshal(body, &meta)
			if meta.Version != backup.Version {
				t.Errorf("version mismatch")
			}
			if meta.CreatedAt.IsZero() {
				meta.CreatedAt = time.Now()
			}
		}
	}
	if !foundMeta {
		t.Error("metadata.json missing")
	}
}
