//go:build darwin

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMacDataDirCandidatesOrder(t *testing.T) {
	home := t.TempDir()
	got := macDataDirCandidates(home)
	wantPrefix := []string{
		filepath.Join(home, "Library", "Application Support", "Termius"),
		filepath.Join(home, "Library", "Application Support", "Termius (MAS)"),
		filepath.Join(home, "Library", "Containers", masBundleID, "Data", "Library", "Application Support", "Termius"),
		filepath.Join(home, "Library", "Containers", masBundleID, "Data", "Library", "Application Support", "Termius (MAS)"),
	}
	if len(got) < len(wantPrefix) {
		t.Fatalf("got %d candidates, want at least %d: %v", len(got), len(wantPrefix), got)
	}
	for i, w := range wantPrefix {
		if got[i] != w {
			t.Errorf("cand[%d]=\n  %s\nwant %s", i, got[i], w)
		}
	}
}

func TestMacDataDirCandidatesScansOtherBundles(t *testing.T) {
	home := t.TempDir()
	extraID := "com.example.termius.old"
	if err := os.MkdirAll(filepath.Join(home, "Library", "Containers", extraID), 0700); err != nil {
		t.Fatal(err)
	}
	got := macDataDirCandidates(home)
	extra := filepath.Join(home, "Library", "Containers", extraID, "Data", "Library", "Application Support", "Termius")
	for _, c := range got {
		if c == extra {
			return
		}
	}
	t.Fatalf("missing scanned bundle path %s in %v", extra, got)
}

func TestPickMacDataDirPrefersIndexedDBOverEmptyDirect(t *testing.T) {
	home := t.TempDir()
	direct := filepath.Join(home, "Library", "Application Support", "Termius")
	mas := filepath.Join(home, "Library", "Containers", masBundleID, "Data", "Library", "Application Support", "Termius")
	if err := os.MkdirAll(direct, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(mas, "IndexedDB"), 0700); err != nil {
		t.Fatal(err)
	}
	got := pickMacDataDir(home, true)
	if got != mas {
		t.Fatalf("got %s, want %s", got, mas)
	}
}

func TestPickMacDataDirDirectWhenHasData(t *testing.T) {
	home := t.TempDir()
	direct := filepath.Join(home, "Library", "Application Support", "Termius")
	if err := os.MkdirAll(filepath.Join(direct, "Local Storage", "leveldb"), 0700); err != nil {
		t.Fatal(err)
	}
	got := pickMacDataDir(home, false)
	if got != direct {
		t.Fatalf("got %s, want %s", got, direct)
	}
}

func TestPickMacDataDirPrefersDirectWhenBothHaveData(t *testing.T) {
	home := t.TempDir()
	direct := filepath.Join(home, "Library", "Application Support", "Termius")
	mas := filepath.Join(home, "Library", "Containers", masBundleID, "Data", "Library", "Application Support", "Termius")
	if err := os.MkdirAll(filepath.Join(direct, "IndexedDB"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(mas, "IndexedDB"), 0700); err != nil {
		t.Fatal(err)
	}
	got := pickMacDataDir(home, true)
	if got != direct {
		t.Fatalf("got %s, want %s", got, direct)
	}
}

func TestPickMacDataDirDefaultMAS(t *testing.T) {
	home := t.TempDir()
	got := pickMacDataDir(home, true)
	want := filepath.Join(home, "Library", "Containers", masBundleID, "Data", "Library", "Application Support", "Termius")
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestPickMacDataDirDefaultDirect(t *testing.T) {
	home := t.TempDir()
	got := pickMacDataDir(home, false)
	want := filepath.Join(home, "Library", "Application Support", "Termius")
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestPickMacDataDirMASNameVariant(t *testing.T) {
	home := t.TempDir()
	mas := filepath.Join(home, "Library", "Containers", masBundleID, "Data", "Library", "Application Support", "Termius (MAS)")
	if err := os.MkdirAll(mas, 0700); err != nil {
		t.Fatal(err)
	}
	got := pickMacDataDir(home, true)
	if got != mas {
		t.Fatalf("got %s, want %s", got, mas)
	}
}

func TestLooksLikeTermiusData(t *testing.T) {
	dir := t.TempDir()
	if looksLikeTermiusData(dir) {
		t.Fatal("empty dir should not look like Termius data")
	}
	if err := os.MkdirAll(filepath.Join(dir, "Local Storage", "leveldb"), 0700); err != nil {
		t.Fatal(err)
	}
	if !looksLikeTermiusData(dir) {
		t.Fatal("expected Local Storage/leveldb to match")
	}
}

func TestKeychainWriteService(t *testing.T) {
	masDir := "/Users/x/Library/Containers/com.termius.mac/Data/Library/Application Support/Termius"
	if got := keychainWriteService(masDir, false); got != keychainServiceMAS {
		t.Fatalf("container path: got %s", got)
	}
	direct := "/Users/x/Library/Application Support/Termius"
	if got := keychainWriteService(direct, true); got != keychainServiceMAS {
		t.Fatalf("mas installed: got %s", got)
	}
	if got := keychainWriteService(direct, false); got != keychainServiceDirect {
		t.Fatalf("direct: got %s", got)
	}
}
