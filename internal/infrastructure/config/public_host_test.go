package config

import "testing"

func TestGatewayPublicHost(t *testing.T) {
	for _, host := range []string{"", "26.31.111.123", "127.0.0.1", "192.168.2.9", "invalid", "26.31.111.123:1935"} {
		t.Run(host, func(t *testing.T) {
			t.Setenv("MCGAME_GATEWAY_PUBLIC_HOST", host)
			cfg, err := Load()
			invalid := host == "invalid" || host == "26.31.111.123:1935"
			if (err != nil) != invalid {
				t.Fatalf("Load error = %v", err)
			}
			if !invalid && cfg.Gateway.PublicHost != host {
				t.Fatalf("PublicHost = %q", cfg.Gateway.PublicHost)
			}
		})
	}
}
