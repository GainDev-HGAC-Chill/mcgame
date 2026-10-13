package registration

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

type Server struct {
	server *http.Server
}

func NewServer(address string, registerer Registerer) *Server {
	return &Server{server: &http.Server{
		Addr: address, Handler: NewHandler(registerer),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8 * 1024,
	}}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return err
	}
	return s.Serve(listener)
}

func (s *Server) Serve(listener net.Listener) error {
	err := s.server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Stop(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		_ = s.server.Close()
		return err
	}
	return nil
}
