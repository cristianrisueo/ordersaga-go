// Servidor HTTP con apagado ordenado: sirve un handler hasta la señal de parada

package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// readHeaderTimeout es el tiempo máximo para leer las cabeceras de una petición.
// shutdownTimeout es el tiempo que se espera a las peticiones en curso al parar.
const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// Run sirve handler en addr hasta la señal de parada y después espera a las peticiones en curso.
func Run(ctx context.Context, addr string, handler http.Handler) error {
	// Arranca el servidor en una goroutine que devuelve su error por un canal
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()

	slog.Info("api listening", "addr", addr)

	// Espera a que el servidor falle o a la señal de parada
	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	case <-ctx.Done():
	}

	// Con la señal, deja de aceptar peticiones y espera a las que están en curso
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http: %w", err)
	}

	// Devuelve nil tras un apagado limpio
	return nil
}
