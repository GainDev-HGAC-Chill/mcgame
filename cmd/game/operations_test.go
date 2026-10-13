package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderClientConfig(t *testing.T) {
	template := []byte("<resource>http://localhost:8888/s/</resource><logic>127.0.0.1:1935/master/test/</logic>")
	got := string(renderClientConfig(template, "192.168.2.9"))
	if strings.Contains(got, "localhost") || strings.Contains(got, "127.0.0.1") || !strings.Contains(got, "192.168.2.9:1935/master/test/") {
		t.Fatal(got)
	}
}

func TestRejectUnsafeLANAddresses(t *testing.T) {
	for _, value := range []string{"", "0.0.0.0", "127.0.0.1", "8.8.8.8", "26.31.111.123", "::1", "invalid"} {
		if assignedPrivateIPv4(value) {
			t.Fatalf("accepted unsafe LAN address %q", value)
		}
	}
}

func TestSetupPreservesExistingDataAndSecrets(t *testing.T) {
	for _, database := range []bool{false, true} {
		t.Run(map[bool]string{false: "secrets", true: "database"}[database], func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".env")
			original := []byte("POSTGRES_PASSWORD=preserve-test-value\n")
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
			if database {
				data := filepath.Join(dir, "volumes", "db", "data")
				if err := os.MkdirAll(data, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(data, "PG_VERSION"), []byte("17"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			s := &stack{dir: dir, envPath: path, assumeYes: true}
			if err := s.setup(); err == nil {
				t.Fatal("setup must refuse existing state even with -y")
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != string(original) {
				t.Fatal("existing secrets were changed")
			}
		})
	}
}

func TestSeedOnlyFreshDatabase(t *testing.T) {
	cases := []struct {
		name, history, applied, tables string
		seed, fail                     bool
	}{
		{"fresh", "false", "0", "false", true, false},
		{"empty history", "true", "0", "false", true, false},
		{"existing", "true", "12", "false", false, false},
		{"missing history with data", "false", "0", "true", false, true},
		{"empty history with data", "true", "0", "true", false, true},
		{"psql short boolean", "t", "0", "false", false, true},
		{"query failure", "", "0", "false", false, true},
		{"invalid count", "true", "bad", "false", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seed, err := seedFreshDatabase(tc.history, tc.applied, tc.tables)
			if seed != tc.seed || (err != nil) != tc.fail {
				t.Fatalf("seed=%v error=%v", seed, err)
			}
		})
	}
}
