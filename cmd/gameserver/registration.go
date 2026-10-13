package main

import (
	"fmt"
	"net"

	"go.uber.org/zap"
	appauth "mcgame-server/internal/application/auth"
	"mcgame-server/internal/infrastructure/config"
	"mcgame-server/internal/presentation/http/registration"
)

func startRegistration(cfg *config.Config, mode config.ServerMode, authService *appauth.Service, log *zap.Logger, serverErr chan<- error) *registration.Server {
	if !cfg.Registration.Enabled || mode != config.ModeMonolith {
		return nil
	}
	server := registration.NewServer(cfg.Registration.Address(), authService)
	listener, err := net.Listen("tcp", cfg.Registration.Address())
	if err != nil {
		serverErr <- fmt.Errorf("registration listener: %w", err)
		return nil
	}
	go func() {
		if err := server.Serve(listener); err != nil {
			serverErr <- fmt.Errorf("registration server: %w", err)
		}
	}()
	log.Info("Registration listener started", zap.String("address", cfg.Registration.Address()))
	return server
}
