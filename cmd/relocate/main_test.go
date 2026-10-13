package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyVerifiedPreservesSourceAndRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	source, target := filepath.Join(dir, "source"), filepath.Join(dir, "target")
	data := []byte("preserve project data")
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if digest, err := copyVerified(source, target, int64(len(data))); err != nil || len(digest) != 64 {
		t.Fatalf("digest=%s err=%v", digest, err)
	}
	for _, path := range []string{source, target} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != string(data) {
			t.Fatalf("copy/source changed: %s %v", path, err)
		}
	}
	if _, err := copyVerified(source, target, int64(len(data))); err == nil {
		t.Fatal("existing destination accepted")
	}
}

func TestCopyVerifiedRejectsChangedSize(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.WriteFile(source, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := copyVerified(source, filepath.Join(dir, "target"), 3); err == nil {
		t.Fatal("size mismatch accepted")
	}
	if data, err := os.ReadFile(source); err != nil || string(data) != "data" {
		t.Fatal("source not preserved")
	}
}
