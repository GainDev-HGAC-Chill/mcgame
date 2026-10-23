// Safe compose database backup and restore operations.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type databaseBackup struct {
	path    string
	size    int64
	modTime time.Time
}

func (s *stack) databaseBackupDir() string {
	return filepath.Join(s.root, ".local", "backups", "database")
}

func (s *stack) publishedDatabaseBackupPath() string {
	return filepath.Join(s.root, "supabase", "backups", "mcgame-live.dump")
}

func databaseBackupName(now time.Time) string {
	return "mcgame-" + now.Format("20060102-150405") + ".dump"
}

func (s *stack) databaseName() (string, error) {
	env, err := loadEnv(s.envPath)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(env.get("POSTGRES_DB"))
	if name == "" {
		return "", fmt.Errorf("missing POSTGRES_DB in %s", s.envPath)
	}
	return name, nil
}

func (s *stack) databaseRunning() bool {
	services, err := s.composeOutput("ps", "--status", "running", "--services")
	if err != nil {
		return false
	}
	for _, service := range strings.Fields(services) {
		if service == "db" {
			return true
		}
	}
	return false
}

func nextBackupPath(dir string, now time.Time) string {
	base := strings.TrimSuffix(databaseBackupName(now), ".dump")
	path := filepath.Join(dir, base+".dump")
	for index := 2; ; index++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
		path = filepath.Join(dir, base+"-"+strconv.Itoa(index)+".dump")
	}
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *stack) backupDatabase() (string, error) {
	return s.createDatabaseBackup(true)
}

func databaseDumpArgs(database, containerPath string) []string {
	return []string{"exec", "-T", "db", "pg_dump", "-U", "supabase_admin", "-d", database, "--format=custom", "--file", containerPath}
}

func databaseRestoreArgs(database, containerPath string) []string {
	return []string{"exec", "-T", "db", "pg_restore", "--exit-on-error", "-U", "supabase_admin", "-d", database, containerPath}
}

func (s *stack) createDatabaseBackup(publish bool) (string, error) {
	if err := dockerReady(); err != nil {
		return "", fmt.Errorf("Docker engine unavailable: %w", err)
	}
	if !s.databaseRunning() {
		return "", fmt.Errorf("database is not running; start the server before creating a backup")
	}
	database, err := s.databaseName()
	if err != nil {
		return "", err
	}
	dir := s.databaseBackupDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	target := nextBackupPath(dir, time.Now())
	partial := target + ".partial"
	containerPath := "/tmp/" + filepath.Base(target)
	defer os.Remove(partial)
	defer func() {
		_ = s.compose("exec", "-T", "db", "rm", "-f", containerPath)
	}()

	fmt.Println("===> Backing up database to", target)
	if err := s.compose(databaseDumpArgs(database, containerPath)...); err != nil {
		return "", fmt.Errorf("pg_dump failed: %w", err)
	}
	if _, err := s.composeOutput("exec", "-T", "db", "pg_restore", "--list", containerPath); err != nil {
		return "", fmt.Errorf("backup archive verification failed: %w", err)
	}
	if err := s.compose("cp", "db:"+containerPath, partial); err != nil {
		return "", fmt.Errorf("copy backup to host: %w", err)
	}
	info, err := os.Stat(partial)
	if err != nil || info.Size() == 0 {
		return "", fmt.Errorf("backup archive is empty or unreadable")
	}
	if err := os.Rename(partial, target); err != nil {
		return "", err
	}
	digest, err := fileSHA256(target)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(target+".sha256", []byte(digest+"  "+filepath.Base(target)+"\n"), 0o600); err != nil {
		return "", err
	}
	fmt.Printf("Database backup verified: %s (%d bytes)\n", target, info.Size())
	if publish {
		published, err := s.publishDatabaseBackup(target, digest)
		if err != nil {
			return "", err
		}
		fmt.Println("Git snapshot updated:", published)
	}
	return target, nil
}

func copyFileAtomic(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	partial := target + ".partial"
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(partial, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		_ = os.Remove(partial)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(partial)
		return closeErr
	}
	if err := os.Rename(partial, target); err != nil {
		_ = os.Remove(partial)
		return err
	}
	return nil
}

func (s *stack) publishDatabaseBackup(source, digest string) (string, error) {
	target := s.publishedDatabaseBackupPath()
	if err := copyFileAtomic(source, target); err != nil {
		return "", fmt.Errorf("publish Git database snapshot: %w", err)
	}
	publishedDigest, err := fileSHA256(target)
	if err != nil || publishedDigest != digest {
		_ = os.Remove(target)
		return "", fmt.Errorf("published Git database snapshot checksum mismatch")
	}
	if err := os.WriteFile(target+".sha256", []byte(digest+"  "+filepath.Base(target)+"\n"), 0o600); err != nil {
		return "", err
	}
	return target, nil
}

func verifyBackupChecksum(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	var header [64]byte
	count, readErr := file.Read(header[:])
	_ = file.Close()
	if readErr != nil && readErr != io.EOF {
		return readErr
	}
	if strings.HasPrefix(string(header[:count]), "version https://git-lfs.github.com/spec/") {
		return fmt.Errorf("database snapshot is a Git LFS pointer; run git lfs pull")
	}
	sidecar, err := os.ReadFile(path + ".sha256")
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	fields := strings.Fields(string(sidecar))
	if len(fields) == 0 {
		return fmt.Errorf("database backup checksum file is empty")
	}
	actual, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(fields[0], actual) {
		return fmt.Errorf("database backup checksum mismatch")
	}
	return nil
}

func listDatabaseBackups(dir string) ([]databaseBackup, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	backups := make([]databaseBackup, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".dump") {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.Size() == 0 {
			continue
		}
		backups = append(backups, databaseBackup{path: filepath.Join(dir, entry.Name()), size: info.Size(), modTime: info.ModTime()})
	}
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})
	return backups, nil
}

func backupWithinDirs(dirs []string, selected string) (string, error) {
	if !filepath.IsAbs(selected) {
		for _, dir := range dirs {
			candidate := filepath.Join(dir, selected)
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
				selected = candidate
				break
			}
		}
	}
	absSelected, err := filepath.Abs(selected)
	if err != nil {
		return "", err
	}
	allowed := false
	for _, dir := range dirs {
		absDir, absErr := filepath.Abs(dir)
		if absErr != nil {
			return "", absErr
		}
		relative, relErr := filepath.Rel(absDir, absSelected)
		if relErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("restore file must be inside an approved database backup directory")
	}
	if !strings.EqualFold(filepath.Ext(absSelected), ".dump") {
		return "", fmt.Errorf("restore file must use the .dump extension")
	}
	info, err := os.Stat(absSelected)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return "", fmt.Errorf("restore file is missing or empty")
	}
	return absSelected, nil
}

func (s *stack) selectDatabaseBackup() (string, error) {
	localDir := s.databaseBackupDir()
	publishedDir := filepath.Dir(s.publishedDatabaseBackupPath())
	dirs := []string{localDir, publishedDir}
	if selected := strings.TrimSpace(os.Getenv("MCGAME_RESTORE_FILE")); selected != "" {
		return backupWithinDirs(dirs, selected)
	}
	backups, err := listDatabaseBackups(localDir)
	if err != nil {
		return "", err
	}
	published, err := listDatabaseBackups(publishedDir)
	if err != nil {
		return "", err
	}
	backups = append(backups, published...)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.After(backups[j].modTime)
	})
	if len(backups) == 0 {
		return "", fmt.Errorf("no database backups found in %s or %s", localDir, publishedDir)
	}
	fmt.Println("Available database backups:")
	for index, backup := range backups {
		fmt.Printf("%d. %s (%d bytes, %s)\n", index+1, filepath.Base(backup.path), backup.size, backup.modTime.Format("2006-01-02 15:04:05"))
	}
	if info, err := os.Stdin.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return "", fmt.Errorf("interactive backup selection unavailable; set MCGAME_RESTORE_FILE")
	}
	fmt.Print("Select backup number: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	choice, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || choice < 1 || choice > len(backups) {
		return "", fmt.Errorf("invalid backup selection")
	}
	return backups[choice-1].path, nil
}

func (s *stack) stop() error {
	backup, err := s.backupDatabase()
	if err != nil {
		return fmt.Errorf("server remains running because database backup failed: %w", err)
	}
	fmt.Println("Backup completed before stop:", backup)
	return s.compose("--profile", "*", "stop", "--timeout", "60")
}

func (s *stack) restoreDatabase() error {
	archive, err := s.selectDatabaseBackup()
	if err != nil {
		return err
	}
	if err := verifyBackupChecksum(archive); err != nil {
		return err
	}
	if err := s.confirm("Restore " + filepath.Base(archive) + "? The current database will be backed up first."); err != nil {
		return err
	}
	if err := dockerReady(); err != nil {
		return fmt.Errorf("Docker engine unavailable: %w", err)
	}
	if _, err := os.Stat(s.envPath); os.IsNotExist(err) {
		if err := s.setup(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if !s.databaseRunning() {
		if err := s.waitForDatabase(); err != nil {
			return err
		}
	}
	safetyBackup, err := s.createDatabaseBackup(false)
	if err != nil {
		return fmt.Errorf("restore aborted because safety backup failed: %w", err)
	}
	fmt.Println("Safety backup:", safetyBackup)
	if err := s.compose("--profile", "*", "stop", "--timeout", "60"); err != nil {
		return err
	}
	if err := s.waitForDatabase(); err != nil {
		return err
	}
	database, err := s.databaseName()
	if err != nil {
		return err
	}
	containerPath := "/tmp/restore-" + filepath.Base(archive)
	defer func() {
		_ = s.compose("exec", "-T", "db", "rm", "-f", containerPath)
	}()
	if err := s.compose("cp", archive, "db:"+containerPath); err != nil {
		return fmt.Errorf("copy restore archive into database container: %w", err)
	}
	if _, err := s.composeOutput("exec", "-T", "db", "pg_restore", "--list", containerPath); err != nil {
		return fmt.Errorf("restore archive verification failed: %w", err)
	}
	if err := s.compose("exec", "-T", "db", "dropdb", "--force", "--if-exists", "--maintenance-db=template1", "-U", "supabase_admin", database); err != nil {
		return fmt.Errorf("drop current database: %w", err)
	}
	if err := s.compose("exec", "-T", "db", "createdb", "--maintenance-db=template1", "-U", "supabase_admin", "-O", "postgres", database); err != nil {
		return fmt.Errorf("recreate database: %w; safety backup: %s", err, safetyBackup)
	}
	if err := s.compose(databaseRestoreArgs(database, containerPath)...); err != nil {
		return fmt.Errorf("restore failed; services remain stopped and safety backup is %s: %w", safetyBackup, err)
	}
	fmt.Println("Database restored from", archive)
	return s.start()
}
