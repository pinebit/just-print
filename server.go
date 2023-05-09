package main

import (
	"context"
	"net"
	"net/http"

	"go.uber.org/zap"
)

type server struct {
	httpServer *http.Server
	logger     *zap.SugaredLogger
}

func NewHttpServer(httpServer *http.Server, logger *zap.SugaredLogger) *server {
	return &server{
		httpServer: httpServer,
		logger:     logger,
	}
}

func (s *server) Start(context.Context) error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return err
	}
	s.logger.Infof("Listening on port: %s", s.httpServer.Addr)

	go func() {
		err := s.httpServer.Serve(ln)
		if err != http.ErrServerClosed {
			s.logger.Fatalw("HTTP server error", "err", err)
		}
	}()
	return nil
}

func (s *server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
