package httpapi

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Server struct {
	server          *http.Server
	shutdownTimeout time.Duration
}

func NewServer(addr string, handler http.Handler, shutdownTimeout time.Duration) *Server {
	if shutdownTimeout == 0 {
		shutdownTimeout = time.Second
	}
	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		shutdownTimeout: shutdownTimeout,
	}
}

func (s *Server) Run(ctx context.Context) error {
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
