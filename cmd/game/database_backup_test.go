package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestDatabaseBackupName(t *testing.T) {
	got := databaseBackupName(time.Date(2026, 10, 23, 21, 5, 9, 0, time.Local))
	if got != "mcgame-20261023-210509.dump" {
		t.Fatalf("name=%q", got)
	}
}

func TestListDatabaseBackupsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "older.dump")
	newer := filepath.Join(dir, "newer.dump")
	for _, path := range []string{older, newer} {
		if err := os.WriteFile(path, []byte("archive"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(older, time.Unix(100, 0), time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, time.Unix(200, 0), time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored.sql"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	backups, err := listDatabaseBackups(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 2 || backups[0].path != newer || backups[1].path != older {
		t.Fatalf("backups=%+v", backups)
	}
}

func TestBackupWithinApprovedDirectories(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, "local")
	published := filepath.Join(root, "published")
	outside := filepath.Join(root, "outside.dump")
	for _, dir := range []string{local, published} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	approved := filepath.Join(published, "mcgame-live.dump")
	for _, path := range []string{approved, outside} {
		if err := os.WriteFile(path, []byte("archive"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := backupWithinDirs([]string{local, published}, approved)
	if err != nil || got != approved {
		t.Fatalf("got=%q err=%v", got, err)
	}
	if _, err := backupWithinDirs([]string{local, published}, outside); err == nil {
		t.Fatal("outside archive was accepted")
	}
}

func TestCopyFileAtomicAndSHA256(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.dump")
	target := filepath.Join(dir, "nested", "target.dump")
	if err := os.WriteFile(source, []byte("database archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFileAtomic(source, target); err != nil {
		t.Fatal(err)
	}
	digest, err := fileSHA256(target)
	if err != nil {
		t.Fatal(err)
	}
	if digest != "658a2dfab5c48bc2e39212657538a1c15cc7f6c8bdb9344bc8b20a3ad4bb1b20" {
		t.Fatalf("digest=%q", digest)
	}
}

func TestVerifyBackupChecksumRejectsMismatchAndLFSPointer(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "backup.dump")
	if err := os.WriteFile(archive, []byte("archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive+".sha256", []byte("bad  backup.dump\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyBackupChecksum(archive); err == nil {
		t.Fatal("checksum mismatch was accepted")
	}
	if err := os.WriteFile(archive, []byte("version https://git-lfs.github.com/spec/v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyBackupChecksum(archive); err == nil {
		t.Fatal("Git LFS pointer was accepted")
	}
}

func TestDatabaseArchiveCommandsPreserveSupabaseOwnershipAndPrivileges(t *testing.T) {
	dump := databaseDumpArgs("postgres", "/tmp/backup.dump")
	restore := databaseRestoreArgs("postgres", "/tmp/backup.dump")
	for _, args := range [][]string{dump, restore} {
		if !slices.Contains(args, "supabase_admin") {
			t.Fatalf("missing Supabase superuser: %v", args)
		}
		if slices.Contains(args, "--no-owner") || slices.Contains(args, "--no-privileges") {
			t.Fatalf("archive command strips ownership or privileges: %v", args)
		}
	}
}
