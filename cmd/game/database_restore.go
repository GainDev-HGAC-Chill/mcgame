// Full database restore orchestration with Supabase background-worker isolation.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type pgNetRestoreState struct {
	enabled          bool
	originalDatabase string
}

func quoteSQLLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func (s *stack) adminQuery(database, query string) (string, error) {
	return s.composeOutput("exec", "-T", "db", "psql", "-v", "ON_ERROR_STOP=1", "-U", "supabase_admin", "-d", database, "-X", "-A", "-t", "-c", query)
}

func (s *stack) adminExec(database, query string) error {
	return s.compose("exec", "-T", "db", "psql", "-v", "ON_ERROR_STOP=1", "-U", "supabase_admin", "-d", database, "-X", "-c", query)
}

func pgNetDatabaseSQL(database string) string {
	return "ALTER SYSTEM SET pg_net.database_name = " + quoteSQLLiteral(database)
}

func terminateDatabaseConnectionsSQL(database string) string {
	return "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = " + quoteSQLLiteral(database) + " AND pid <> pg_backend_pid()"
}

func databaseConnectionCountSQL(database string) string {
	return "SELECT count(*) FROM pg_stat_activity WHERE datname = " + quoteSQLLiteral(database)
}

func (s *stack) waitForNoDatabaseConnections(database string) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		value, err := s.adminQuery("template1", databaseConnectionCountSQL(database))
		if err == nil && strings.TrimSpace(value) == "0" {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("database %s still has active connections after pg_net isolation", database)
}

func (s *stack) pausePGNetForRestore(database string) (pgNetRestoreState, error) {
	present, err := s.adminQuery(database, "SELECT to_regprocedure('net.worker_restart()') IS NOT NULL")
	if err != nil {
		return pgNetRestoreState{}, err
	}
	if strings.TrimSpace(present) != "t" {
		return pgNetRestoreState{}, nil
	}
	original, err := s.adminQuery(database, "SELECT current_setting('pg_net.database_name', true)")
	if err != nil {
		return pgNetRestoreState{}, err
	}
	original = strings.TrimSpace(original)
	if original == "" {
		original = database
	}
	state := pgNetRestoreState{enabled: true, originalDatabase: original}
	rollback := func() {
		_ = s.adminExec("template1", pgNetDatabaseSQL(original))
		_ = s.adminExec("template1", "SELECT pg_reload_conf()")
		_ = s.adminExec(database, "SELECT net.worker_restart()")
	}
	if err := s.adminExec(database, pgNetDatabaseSQL("template1")); err != nil {
		return pgNetRestoreState{}, err
	}
	if err := s.adminExec(database, "SELECT net.worker_restart()"); err != nil {
		rollback()
		return pgNetRestoreState{}, err
	}
	if err := s.adminExec("template1", terminateDatabaseConnectionsSQL(database)); err != nil {
		rollback()
		return state, err
	}
	if err := s.waitForNoDatabaseConnections(database); err != nil {
		rollback()
		return state, err
	}
	return state, nil
}

func (s *stack) resumePGNetAfterRestore(state pgNetRestoreState, database string) error {
	if !state.enabled {
		return nil
	}
	if err := s.adminExec("template1", pgNetDatabaseSQL(state.originalDatabase)); err != nil {
		return err
	}
	if err := s.adminExec("template1", "SELECT pg_reload_conf()"); err != nil {
		return err
	}
	if err := s.adminExec(database, "SELECT net.worker_restart()"); err != nil {
		return err
	}
	configured, err := s.adminQuery(database, "SELECT current_setting('pg_net.database_name', true)")
	if err != nil {
		return err
	}
	if strings.TrimSpace(configured) != state.originalDatabase {
		return fmt.Errorf("pg_net resumed with database %q instead of %q", strings.TrimSpace(configured), state.originalDatabase)
	}
	return nil
}

func (s *stack) expectedMigrationCount() (int, error) {
	entries, err := os.ReadDir(filepath.Join(s.root, "supabase", "migrations"))
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".sql") {
			count++
		}
	}
	return count, nil
}

func restoredDatabaseAuditSQL() string {
	return "SELECT (SELECT count(*) FROM supabase_migrations.schema_migrations), (SELECT count(*) FROM data.data_tbl_item_template), coalesce((SELECT tableowner FROM pg_tables WHERE schemaname='vault' AND tablename='secrets'),''), coalesce((SELECT tableowner FROM pg_tables WHERE schemaname='player' AND tablename='characters'),''), coalesce((SELECT tableowner FROM pg_tables WHERE schemaname='auth' AND tablename='users'),'')"
}

func validateRestoredDatabaseAudit(value string, expectedMigrations int) error {
	parts := strings.Split(strings.TrimSpace(value), "|")
	if len(parts) != 5 {
		return fmt.Errorf("unexpected restored database audit result %q", value)
	}
	migrations, migrationErr := strconv.Atoi(parts[0])
	items, itemErr := strconv.ParseInt(parts[1], 10, 64)
	if migrationErr != nil || migrations != expectedMigrations {
		return fmt.Errorf("restored migration history has %q rows, want %d", parts[0], expectedMigrations)
	}
	if itemErr != nil || items <= 0 {
		return fmt.Errorf("restored item catalog is empty")
	}
	if parts[2] != "supabase_admin" || parts[3] != "postgres" || parts[4] != "supabase_auth_admin" {
		return fmt.Errorf("restored ownership mismatch: vault=%q player=%q auth=%q", parts[2], parts[3], parts[4])
	}
	return nil
}

func (s *stack) waitForRestoredDatabase(database string) error {
	expectedMigrations, err := s.expectedMigrationCount()
	if err != nil {
		return err
	}
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		value, queryErr := s.adminQuery(database, restoredDatabaseAuditSQL())
		if queryErr == nil {
			if auditErr := validateRestoredDatabaseAudit(value, expectedMigrations); auditErr == nil {
				return nil
			} else {
				lastErr = auditErr
			}
		} else {
			lastErr = queryErr
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("restored database did not pass verification: %w", lastErr)
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
	pgNetState, err := s.pausePGNetForRestore(database)
	if err != nil {
		return fmt.Errorf("pause pg_net before restore: %w", err)
	}
	pgNetResumed := false
	defer func() {
		if !pgNetResumed {
			_ = s.resumePGNetAfterRestore(pgNetState, database)
		}
	}()
	if err := s.compose("exec", "-T", "db", "dropdb", "--force", "--if-exists", "--maintenance-db=template1", "-U", "supabase_admin", database); err != nil {
		return fmt.Errorf("drop current database: %w", err)
	}
	if err := s.compose("exec", "-T", "db", "createdb", "--maintenance-db=template1", "-U", "supabase_admin", "-O", "postgres", database); err != nil {
		return fmt.Errorf("recreate database: %w; safety backup: %s", err, safetyBackup)
	}
	if err := s.compose(databaseRestoreArgs(database, containerPath)...); err != nil {
		return fmt.Errorf("restore failed; services remain stopped and safety backup is %s: %w", safetyBackup, err)
	}
	if err := s.waitForRestoredDatabase(database); err != nil {
		return err
	}
	if err := s.resumePGNetAfterRestore(pgNetState, database); err != nil {
		return fmt.Errorf("resume pg_net after restore: %w", err)
	}
	pgNetResumed = true
	fmt.Println("Database restored and verified from", archive)
	return s.start()
}
