// Radmin is opt-in and restricted to an active Radmin adapter, not arbitrary public IPs.
package main

import (
	"fmt"
	"net"
	"strings"
)

func radminAddress(iface net.Interface, ip net.IP) bool {
	if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
		return false
	}
	if !strings.Contains(strings.ToLower(iface.Name), "radmin") {
		return false
	}
	v4 := ip.To4()
	return v4 != nil && v4[0] == 26
}

func selectRadminIPv4(requested string, candidates []string) (string, error) {
	if requested != "" {
		ip := net.ParseIP(requested)
		if ip != nil && ip.To4() != nil {
			for _, candidate := range candidates {
				if ip.Equal(net.ParseIP(candidate)) {
					return candidate, nil
				}
			}
		}
		return "", fmt.Errorf("MCGAME_RADMIN_IP must be a 26.x.x.x IPv4 assigned to an active Radmin adapter")
	}
	if len(candidates) != 1 {
		return "", fmt.Errorf("expected one active Radmin IPv4, found %d; enable Radmin VPN or set MCGAME_RADMIN_IP explicitly", len(candidates))
	}
	return candidates[0], nil
}

func assignedRadminIPv4(requested string) (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("list Radmin adapters: %w", err)
	}
	var candidates []string
	for _, iface := range interfaces {
		if !strings.Contains(strings.ToLower(iface.Name), "radmin") || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			return "", fmt.Errorf("read Radmin adapter %s: %w", iface.Name, err)
		}
		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err == nil && radminAddress(iface, ip) {
				candidates = append(candidates, ip.String())
			}
		}
	}
	return selectRadminIPv4(requested, candidates)
}
