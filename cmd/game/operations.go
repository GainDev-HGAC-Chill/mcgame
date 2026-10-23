// Safe local/LAN operation helpers. No command in this file deletes persistent data.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func dockerReady() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := attach(exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")).Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("Docker engine did not respond within 15s: %w", ctx.Err())
		}
		return err
	}
	return nil
}

func (s *stack) start() error {
	if err := dockerReady(); err != nil {
		return fmt.Errorf("Docker engine unavailable: start Docker Desktop and check WSL2: %w", err)
	}
	if _, err := exec.LookPath("supabase"); err != nil {
		return fmt.Errorf("Supabase CLI missing; install it on PATH or in bin/: %w", err)
	}
	if _, err := os.Stat(s.envPath); os.IsNotExist(err) {
		if err := s.setup(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := s.up(); err != nil {
		return err
	}
	s.printURLs()
	return s.check()
}

func (s *stack) clientHost() string {
	if env, err := loadEnv(s.envPath); err == nil {
		if host := env.get("GAME_PUBLIC_HOST"); host != "" {
			return host
		}
		if host := env.get("GAME_BIND_ADDRESS"); host != "" {
			return host
		}
	}
	return "127.0.0.1"
}

func clientURL(host string) string {
	return "http://" + host + ":8888/s/sv/GameLoaders.swf?isExpand=true"
}

func (s *stack) printURLs() {
	fmt.Println("Client:", clientURL(s.clientHost()))
	fmt.Println("Dashboard (host only): http://127.0.0.1:3000")
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Fprintln(os.Stderr, "network interfaces:", err)
		return
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ip, network, err := net.ParseCIDR(address.String())
			if err == nil && ip.To4() != nil && ip.IsPrivate() {
				fmt.Printf("LAN candidate: %s %s (%s)\n", iface.Name, ip, network)
			}
			if err == nil && radminAddress(iface, ip) {
				fmt.Printf("Radmin candidate: %s %s (%s)\n", iface.Name, ip, network)
			}
		}
	}
}

func assignedPrivateIPv4(value string) bool {
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() == nil || !ip.IsPrivate() || ip.IsLoopback() {
		return false
	}
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, address := range addresses {
		local, _, err := net.ParseCIDR(address.String())
		if err == nil && local.Equal(ip) {
			return true
		}
	}
	return false
}

func renderClientConfig(template []byte, host string) []byte {
	text := strings.ReplaceAll(string(template), "localhost", host)
	return []byte(strings.ReplaceAll(text, "127.0.0.1", host))
}

func (s *stack) configureClient(mode string) error {
	bindHost := "127.0.0.1"
	publicHost := bindHost
	dashboardHost := "127.0.0.2"
	switch mode {
	case "local":
	case "lan":
		bindHost = os.Getenv("MCGAME_LAN_IP")
		if !assignedPrivateIPv4(bindHost) {
			return fmt.Errorf("set MCGAME_LAN_IP to an assigned private IPv4 address; game status lists candidates")
		}
		publicHost = bindHost
		dashboardHost = bindHost
	case "radmin":
		var err error
		bindHost, err = assignedRadminIPv4(os.Getenv("MCGAME_RADMIN_IP"))
		if err != nil {
			return err
		}
		publicHost = bindHost
		dashboardHost = bindHost
	case "online":
		publicHost = strings.TrimSpace(os.Getenv("MCGAME_ONLINE_HOST"))
		if !validOnlineHost(publicHost) {
			return fmt.Errorf("set MCGAME_ONLINE_HOST to a public IPv4 address or DNS hostname without scheme, port, or path")
		}
		bindHost = "0.0.0.0"
	default:
		return fmt.Errorf("unknown client profile %q", mode)
	}
	if _, err := os.Stat(s.envPath); os.IsNotExist(err) {
		if err := s.setup(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	env, err := loadEnv(s.envPath)
	if err != nil {
		return fmt.Errorf("configure client: run game setup first: %w", err)
	}
	template, err := os.ReadFile(filepath.Join(s.root, "frontend", "s", "sv", "profile", "config.xml"))
	if err != nil {
		return err
	}
	dir := filepath.Join(s.dir, "client")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, mode+".xml")
	if err := os.WriteFile(path, renderClientConfig(template, publicHost), 0o644); err != nil {
		return err
	}
	env.set("GAME_BIND_ADDRESS", bindHost)
	env.set("GAME_PUBLIC_HOST", publicHost)
	env.set("GAME_CLIENT_CONFIG_PATH", "./client/"+mode+".xml")
	env.set("DASHBOARD_REMOTE_BIND_ADDRESS", dashboardHost)
	env.set("DASHBOARD_ALLOWED_ORIGINS", dashboardAllowedOrigins(dashboardHost))
	if err := env.save(); err != nil {
		return err
	}
	fmt.Println("Client profile:", mode, "(apply with game start)")
	if mode == "online" {
		fmt.Println("WARNING: Online VPS exposes game HTTP/RTMPE publicly. TLS, firewall and security hardening are not configured by this profile.")
	}
	s.printURLs()
	return nil
}

func dashboardAllowedOrigins(remoteHost string) string {
	return "http://localhost:3000,http://127.0.0.1:3000,http://" + remoteHost + ":3000"
}

func validOnlineHost(value string) bool {
	if value == "" || strings.ContainsAny(value, "/:\\ ") {
		return false
	}
	if ip := net.ParseIP(value); ip != nil {
		ip = ip.To4()
		return ip != nil && ip[0] != 26 && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsUnspecified() && !ip.IsMulticast() && !ip.IsLinkLocalUnicast()
	}
	if len(value) > 253 || !strings.Contains(value, ".") {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}

func (s *stack) check() error {
	if err := dockerReady(); err != nil {
		return fmt.Errorf("Docker engine unavailable: %w", err)
	}
	if err := s.compose("ps", "--all"); err != nil {
		return err
	}
	if err := s.compose("exec", "-T", "db", "pg_isready", "-U", "postgres", "-h", "localhost"); err != nil {
		return fmt.Errorf("database readiness failed: %w", err)
	}
	if err := s.compose("exec", "-T", "redis", "redis-cli", "ping"); err != nil {
		return fmt.Errorf("Redis readiness failed: %w", err)
	}
	if err := s.compose("exec", "-T", "rtmp-edge", "nginx", "-t"); err != nil {
		return fmt.Errorf("edge configuration failed: %w", err)
	}
	fmt.Println("Infrastructure checks only; Flash login/gameplay still require real client testing.")
	return nil
}

func (s *stack) openClient() error {
	return s.launchClient(clientURL(s.clientHost()))
}

func (s *stack) launchClient(url string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("open the Flash projector manually with %s", url)
	}
	path := filepath.Join(s.root, "docs", "flashplayer", "flashplayer_32_sa_debug.exe")
	cmd := exec.Command(path, url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start Flash projector: %w", err)
	}
	fmt.Println("Flash projector launched; login/gameplay not yet verified.")
	return cmd.Process.Release()
}
