package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

type HttpServer struct {
	server          *http.Server
	shutdownTimeout time.Duration
}

func NewHttpServer(addr string, handler http.Handler, shutdownTimeout time.Duration) *HttpServer {
	if shutdownTimeout == 0 {
		shutdownTimeout = time.Second
	}
	return &HttpServer{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		shutdownTimeout: shutdownTimeout,
	}
}

func (s *HttpServer) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		log.Println("closing http server")
	case err := <-errCh:
		return fmt.Errorf("http server closed, cause: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server graceful shutdown failed: %w", err)
	}
	log.Println("http server stopped")
	return nil
}
