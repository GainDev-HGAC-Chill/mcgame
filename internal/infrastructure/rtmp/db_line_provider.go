// Open-sourced by BaoLT

// DB-backed line list provider sourced from the runtime config store.
// Live client counts come from the RTMP connection tracker.
package rtmp

import (
	"context"
	"net"
	"net/url"
	"strings"

	"mcgame-server/internal/infrastructure/config/store"
)

type ConnectionTracker interface {
	GetChannelConnectionCount(channelID int) int
}

type DBLineProvider struct {
	store       *store.Store
	connTracker ConnectionTracker
	publicHost  string
}

func NewDBLineProvider(s *store.Store, connTracker ConnectionTracker, publicHost string) *DBLineProvider {
	return &DBLineProvider{store: s, connTracker: connTracker, publicHost: publicHost}
}

func (p *DBLineProvider) GetLineList(ctx context.Context) ([]map[string]interface{}, error) {
	lines := p.store.GatewayLines()
	out := make([]map[string]interface{}, 0, len(lines))
	for _, line := range lines {
		clients := 0
		if p.connTracker != nil {
			clients = p.connTracker.GetChannelConnectionCount(line.ID)
		}

		out = append(out, map[string]interface{}{
			"id":      line.ID,
			"name":    line.Name,
			"url":     publicLineURL(line.URL, p.publicHost),
			"clients": clients,
			"max":     line.MaxClients,
			"status":  statusCodeFor(line.Status),
			"auction": line.Auction,
			"guild":   line.Guild,
		})
	}
	return out, nil
}

func publicLineURL(value, publicHost string) string {
	if publicHost == "" || net.ParseIP(publicHost) == nil {
		return value
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "rtmp" && u.Scheme != "rtmpe") {
		return value
	}
	ip := net.ParseIP(u.Hostname())
	if !strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback()) {
		return value
	}
	port := u.Port()
	u.Host = net.JoinHostPort(publicHost, port)
	if port == "" {
		u.Host = publicHost
		if strings.Contains(publicHost, ":") {
			u.Host = "[" + publicHost + "]"
		}
	}
	return u.String()
}

func statusCodeFor(status string) int {
	switch status {
	case "offline", "maintenance":
		return 20
	default:
		return 10
	}
}
