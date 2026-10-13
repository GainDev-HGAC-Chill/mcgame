package main

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRadminAddress(t *testing.T) {
	for _, tc := range []struct {
		name, ip string
		flags    net.Flags
		want     bool
	}{
		{"Radmin VPN", "26.31.111.123", net.FlagUp, true},
		{"radmin vpn", "26.1.2.3", net.FlagUp, true},
		{"Ethernet", "26.31.111.123", net.FlagUp, false},
		{"Radmin VPN", "192.168.2.9", net.FlagUp, false},
		{"Radmin VPN", "8.8.8.8", net.FlagUp, false},
		{"Radmin VPN", "::1", net.FlagUp, false},
		{"Radmin VPN", "invalid", net.FlagUp, false},
		{"Radmin VPN", "26.31.111.123", 0, false},
		{"Radmin VPN", "26.31.111.123", net.FlagUp | net.FlagLoopback, false},
	} {
		if got := radminAddress(net.Interface{Name: tc.name, Flags: tc.flags}, net.ParseIP(tc.ip)); got != tc.want {
			t.Fatalf("%+v: got %v", tc, got)
		}
	}
}

func TestSelectRadminIPv4(t *testing.T) {
	for _, tc := range []struct {
		requested  string
		candidates []string
		want       string
	}{
		{"", []string{"26.1.2.3"}, "26.1.2.3"},
		{"26.1.2.3", []string{"26.1.2.3", "26.4.5.6"}, "26.1.2.3"},
		{"", nil, ""},
		{"", []string{"26.1.2.3", "26.4.5.6"}, ""},
		{"26.7.8.9", []string{"26.1.2.3"}, ""},
		{"8.8.8.8", []string{"26.1.2.3"}, ""},
		{"invalid", []string{"26.1.2.3"}, ""},
	} {
		got, err := selectRadminIPv4(tc.requested, tc.candidates)
		if got != tc.want || (err != nil) != (tc.want == "") {
			t.Fatalf("%+v: got %q, error %v", tc, got, err)
		}
	}
}

func TestRenderRadminClientConfig(t *testing.T) {
	template := []byte("<resource>http://localhost:8888/s/</resource><logic>127.0.0.1:1935/master/test/</logic>")
	got := string(renderClientConfig(template, "26.31.111.123"))
	if got != "<resource>http://26.31.111.123:8888/s/</resource><logic>26.31.111.123:1935/master/test/</logic>" {
		t.Fatal(got)
	}
}

func TestRadminRejectDoesNotChangeConfig(t *testing.T) {
	t.Setenv("MCGAME_RADMIN_IP", "8.8.8.8")
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	original := "GAME_BIND_ADDRESS=127.0.0.1\nPOSTGRES_PASSWORD=preserve-test-value\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &stack{root: dir, dir: dir, envPath: path}
	if err := s.configureClient("radmin"); err == nil || !strings.Contains(err.Error(), "Radmin") {
		t.Fatalf("expected Radmin validation failure, got %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != original {
		t.Fatalf("existing configuration changed: %v", err)
	}
}
