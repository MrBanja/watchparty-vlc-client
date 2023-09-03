package http_server

import (
	"context"
	"net/http"
	"sync"
	"time"
)

func Serve(ctx context.Context, server *http.Server, timeout time.Duration, serve func(server *http.Server) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errors := make(chan error, 2)
	defer close(errors)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		if err := serve(server); err != nil {
			if err.Error() != http.ErrServerClosed.Error() {
				errors <- err
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		<-ctx.Done()

		shutdownTimeoutCtx, shutdownTimeoutCtxCancel := context.WithTimeout(context.Background(), timeout)
		defer shutdownTimeoutCtxCancel()

		if err := server.Shutdown(shutdownTimeoutCtx); err != nil {
			errors <- err
		}
	}()

	wg.Wait()

	select {
	case err := <-errors:
		return err
	default:
		return nil
	}
}
