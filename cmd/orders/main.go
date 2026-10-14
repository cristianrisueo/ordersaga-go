// Binario del servicio orders: aplica las migraciones, arranca la API o el productor de la outbox según el modo

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cristianrisueo/ordersaga-go/internal/database"
	"github.com/cristianrisueo/ordersaga-go/internal/migrations"
	"github.com/cristianrisueo/ordersaga-go/internal/orders"
	"github.com/cristianrisueo/ordersaga-go/internal/saga"
	"github.com/jackc/pgx/v5/pgxpool"
)

// apiAddr es la dirección en la que escucha la API.
// readHeaderTimeout es el tiempo máximo para leer las cabeceras de una petición.
// shutdownTimeout es el tiempo que se espera a las peticiones en curso al parar.
const (
	apiAddr           = ":8000"
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// main ejecuta el servicio y sale con código 1 si falla.
func main() {
	// Ejecuta el servicio fuera de main para que sus defer corran siempre antes de salir
	if err := run(); err != nil {
		slog.Error("run orders", "error", err)
		os.Exit(1)
	}
}

// run lee la configuración, conecta con la base de datos y arranca el modo elegido.
func run() error {
	// Configura slog con salida JSON como logger por defecto
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Lee el modo de despliegue del primer argumento
	if len(os.Args) < 2 {
		return errors.New("missing mode: api, producer or migrate")
	}
	mode := os.Args[1]

	// Lee la configuración del entorno: DSN y Kafka
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("missing DATABASE_URL")
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		return errors.New("missing KAFKA_BROKERS")
	}

	brokers := strings.Split(kafkaBrokers, ",")

	// Crea el contexto que se cancela con SIGINT o SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	// Realiza la conexión con la BD
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	defer pool.Close()

	// Arranca el modo elegido (migrate, api o producer)
	switch mode {
	case "migrate":
		return database.Migrate(ctx, pool, migrations.Orders, "orders")
	case "api":
		return runAPI(ctx, pool)
	case "producer":
		return saga.RunProducer(ctx, pool, brokers, "orders")
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
}

// runAPI sirve la API HTTP hasta la señal de parada.
func runAPI(ctx context.Context, pool *pgxpool.Pool) error {
	// Crea el repositorio, el servicio y el handler
	repo := orders.NewRepository(pool)
	service := orders.NewService(pool, repo)
	handler := orders.NewHandler(service)

	// Arranca el servidor en una goroutine que devuelve su error por un canal
	server := &http.Server{
		Addr:              apiAddr,
		Handler:           orders.Routes(handler),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()

	slog.Info("api listening", "addr", apiAddr)

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
