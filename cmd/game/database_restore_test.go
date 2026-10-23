package main

import (
	"strings"
	"testing"
)

func TestRestoreSQLQuotesDatabaseNames(t *testing.T) {
	value := "game'db"
	for _, query := range []string{
		pgNetDatabaseSQL(value),
		terminateDatabaseConnectionsSQL(value),
		databaseConnectionCountSQL(value),
	} {
		if !strings.Contains(query, "'game''db'") {
			t.Fatalf("query does not quote database name: %q", query)
		}
	}
}

func TestValidateRestoredDatabaseAudit(t *testing.T) {
	if err := validateRestoredDatabaseAudit("103|6468|supabase_admin|postgres|supabase_auth_admin", 103); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"102|6468|supabase_admin|postgres|supabase_auth_admin",
		"103|0|supabase_admin|postgres|supabase_auth_admin",
		"103|6468|postgres|postgres|supabase_auth_admin",
		"invalid",
	} {
		if err := validateRestoredDatabaseAudit(value, 103); err == nil {
			t.Fatalf("invalid audit accepted: %q", value)
		}
	}
}

func TestRestoredDatabaseAuditChecksCriticalOwnersAndData(t *testing.T) {
	query := restoredDatabaseAuditSQL()
	for _, required := range []string{
		"supabase_migrations.schema_migrations",
		"data.data_tbl_item_template",
		"schemaname='vault'",
		"schemaname='player'",
		"schemaname='auth'",
	} {
		if !strings.Contains(query, required) {
			t.Fatalf("audit query missing %q: %s", required, query)
		}
	}
}
