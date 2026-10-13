// Relocate an offline directory with verified copies and a Windows junction.
// Existing destinations, links and special files are rejected; failures retain data.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type entry struct {
	Path   string
	Size   int64
	SHA256 string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: relocate <offline-source-directory> <new-destination>")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(source, target string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("Windows only")
	}
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return err
	}
	if filepath.Dir(source) == source || strings.ContainsAny(source+target, "\"&|<>%\r\n") {
		return fmt.Errorf("unsafe source or path characters")
	}
	if strings.HasPrefix(strings.ToLower(target)+"\\", strings.ToLower(source)+"\\") || strings.HasPrefix(strings.ToLower(source)+"\\", strings.ToLower(target)+"\\") {
		return fmt.Errorf("paths overlap")
	}
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("source must be a real directory")
	}
	backup := source + ".relocation-backup"
	for _, path := range []string{target, backup, target + ".manifest.json"} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			return fmt.Errorf("path already exists or inaccessible: %s", path)
		}
	}
	var entries []entry
	var dirs []string
	err = filepath.WalkDir(source, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, rel)
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing special file: %s", path)
		}
		entries = append(entries, entry{Path: rel, Size: info.Size()})
		return nil
	})
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(target, dir), 0o755); err != nil {
			return err
		}
	}
	var total int64
	for i := range entries {
		e := &entries[i]
		digest, err := copyVerified(filepath.Join(source, e.Path), filepath.Join(target, e.Path), e.Size)
		if err != nil {
			return fmt.Errorf("copy %s: %w; source retained", e.Path, err)
		}
		e.SHA256 = digest
		total += e.Size
		if e.Size > 100<<20 {
			fmt.Printf("Verified %s (%d bytes)\n", e.Path, e.Size)
		}
	}
	manifest, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(target+".manifest.json", manifest, 0o600); err != nil {
		return err
	}
	if err := os.Rename(source, backup); err != nil {
		return fmt.Errorf("rename source: %w; both copies retained", err)
	}
	cmd := exec.Command("cmd.exe", "/c", "mklink", "/J", source, target)
	if output, err := cmd.CombinedOutput(); err != nil {
		rollbackErr := os.Rename(backup, source)
		return fmt.Errorf("junction failed: %v (%s); restore result: %v; destination retained", err, output, rollbackErr)
	}
	linkedInfo, linkErr := os.Stat(source)
	targetInfo, targetErr := os.Stat(target)
	if linkErr != nil || targetErr != nil || !os.SameFile(linkedInfo, targetInfo) {
		return fmt.Errorf("junction verification failed: link=%v target=%v; backup retained", linkErr, targetErr)
	}
	for _, e := range entries {
		if err := os.Remove(filepath.Join(backup, e.Path)); err != nil {
			return fmt.Errorf("verified destination active; backup cleanup failed: %w", err)
		}
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := os.Chmod(filepath.Join(backup, dirs[i]), 0o755); err != nil {
			return fmt.Errorf("backup directory permissions: %w", err)
		}
		if err := os.Remove(filepath.Join(backup, dirs[i])); err != nil {
			return fmt.Errorf("backup directory cleanup: %w", err)
		}
	}
	fmt.Printf("PASS: %s -> %s; %d files, %.2f GiB, SHA256 verified\n", source, target, len(entries), float64(total)/(1<<30))
	return nil
}

func copyVerified(source, target string, size int64) (string, error) {
	in, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(out, hash), in)
	syncErr := out.Sync()
	closeErr := out.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if syncErr != nil {
		return "", syncErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if n != size {
		return "", fmt.Errorf("source size changed")
	}
	written, err := os.Open(target)
	if err != nil {
		return "", err
	}
	defer written.Close()
	verify := sha256.New()
	n, err = io.Copy(verify, written)
	if err != nil {
		return "", err
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	if n != size || digest != hex.EncodeToString(verify.Sum(nil)) {
		return "", fmt.Errorf("copy checksum mismatch")
	}
	return digest, nil
}
