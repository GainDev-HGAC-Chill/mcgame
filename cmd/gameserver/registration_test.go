package main

import (
	"net"
	"strconv"
	"testing"

	"go.uber.org/zap"
	"mcgame-server/internal/infrastructure/config"
)

func TestRegistrationDisabledAndLineMode(t *testing.T) {
	for _, tc := range []struct {
		enabled bool
		mode    config.ServerMode
	}{
		{false, config.ModeMonolith},
		{true, config.ModeLine},
	} {
		cfg := &config.Config{Registration: config.RegistrationConfig{Enabled: tc.enabled, Host: "127.0.0.1", Port: 0}}
		errors := make(chan error, 1)
		if server := startRegistration(cfg, tc.mode, nil, zap.NewNop(), errors); server != nil {
			t.Fatal("registration started outside enabled monolith")
		}
		if len(errors) != 0 {
			t.Fatal("disabled listener attempted startup")
		}
	}
}

func TestRegistrationBindErrorParticipatesInLifecycle(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Registration: config.RegistrationConfig{Enabled: true, Host: "127.0.0.1", Port: portNumber}}
	errors := make(chan error, 1)
	if server := startRegistration(cfg, config.ModeMonolith, nil, zap.NewNop(), errors); server != nil {
		t.Fatal("failed listener returned ready server")
	}
	select {
	case err := <-errors:
		if err == nil {
			t.Fatal("nil startup error")
		}
	default:
		t.Fatal("startup failure not sent to shutdown loop")
	}
}
