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

func TestValidateOnlineHost(t *testing.T) {
	for _, value := range []string{"203.0.113.10", "game.example.com", "vps-1.example.net"} {
		if !validOnlineHost(value) {
			t.Fatalf("rejected online host %q", value)
		}
	}
	for _, value := range []string{"", "localhost", "127.0.0.1", "192.168.1.2", "26.31.111.123", "http://game.example.com", "game.example.com:8888", "bad_host.example"} {
		if validOnlineHost(value) {
			t.Fatalf("accepted unsafe online host %q", value)
		}
	}
}

func TestDashboardAllowedOriginsUsesSelectedProfileHost(t *testing.T) {
	got := dashboardAllowedOrigins("26.31.111.123")
	for _, origin := range []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://26.31.111.123:3000"} {
		if !strings.Contains(got, origin) {
			t.Fatalf("missing %q in %q", origin, got)
		}
	}
}

func TestConfigureOnlineProfileSeparatesPublicHostFromBindAddress(t *testing.T) {
	root := t.TempDir()
	dockerDir := filepath.Join(root, "docker")
	templateDir := filepath.Join(root, "frontend", "s", "sv", "profile")
	if err := os.MkdirAll(dockerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(dockerDir, ".env")
	if err := os.WriteFile(envPath, []byte("POSTGRES_DB=postgres\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "config.xml"), []byte("http://localhost:8888 127.0.0.1:1935"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MCGAME_ONLINE_HOST", "game.example.com")
	s := &stack{root: root, dir: dockerDir, envPath: envPath}
	if err := s.configureClient("online"); err != nil {
		t.Fatal(err)
	}
	env, err := loadEnv(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if env.get("GAME_BIND_ADDRESS") != "0.0.0.0" || env.get("GAME_PUBLIC_HOST") != "game.example.com" {
		t.Fatalf("bind=%q public=%q", env.get("GAME_BIND_ADDRESS"), env.get("GAME_PUBLIC_HOST"))
	}
	if env.get("DASHBOARD_REMOTE_BIND_ADDRESS") != "127.0.0.2" {
		t.Fatalf("dashboard bind=%q", env.get("DASHBOARD_REMOTE_BIND_ADDRESS"))
	}
	profile, err := os.ReadFile(filepath.Join(dockerDir, "client", "online.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(profile), "localhost") || !strings.Contains(string(profile), "game.example.com") {
		t.Fatalf("profile=%q", profile)
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

func TestMigrationPushIncludesOutOfOrderMigrationsWithoutReseedingExistingData(t *testing.T) {
	args := migrationPushArgs(`I:\mcgame`, "postgresql://local", false)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--include-all") {
		t.Fatalf("missing --include-all: %v", args)
	}
	if strings.Contains(joined, "--include-seed") {
		t.Fatalf("existing database would be reseeded: %v", args)
	}
	if args[len(args)-1] != "--yes" {
		t.Fatalf("non-interactive confirmation missing: %v", args)
	}
}

func TestBuildCachePruneRemovesAllCache(t *testing.T) {
	args := strings.Join(buildCachePruneArgs(), " ")
	if args != "builder prune --all --force" {
		t.Fatalf("unexpected Docker build cache policy: %s", args)
	}
}
