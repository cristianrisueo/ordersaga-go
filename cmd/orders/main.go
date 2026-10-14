// Binario del servicio orders: aplica las migraciones, arranca la API o el productor de la outbox según el modo

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cristianrisueo/ordersaga-go/internal/platform/database"
	"github.com/cristianrisueo/ordersaga-go/internal/platform/httpserver"
	"github.com/cristianrisueo/ordersaga-go/internal/saga"
	"github.com/cristianrisueo/ordersaga-go/internal/services/orders"
	"github.com/cristianrisueo/ordersaga-go/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// apiAddr es la dirección en la que escucha la API.
const apiAddr = ":8000"

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

	// Sirve la API hasta la señal de parada
	return httpserver.Run(ctx, apiAddr, orders.Routes(handler))
}
