// Binario del servicio payments: aplica las migraciones, arranca la API, el productor o el consumidor según el modo

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/cristianrisueo/ordersaga-go/internal/platform/database"
	"github.com/cristianrisueo/ordersaga-go/internal/platform/httpserver"
	"github.com/cristianrisueo/ordersaga-go/internal/saga"
	"github.com/cristianrisueo/ordersaga-go/internal/services/payments"
	"github.com/cristianrisueo/ordersaga-go/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// apiAddr es la dirección en la que escucha la API.
const apiAddr = ":8000"

// main ejecuta el servicio y sale con código 1 si falla.
func main() {
	// Ejecuta el servicio fuera de main para que sus defer corran siempre antes de salir
	if err := run(); err != nil {
		slog.Error("run payments", "error", err)
		os.Exit(1)
	}
}

// run lee la configuración, conecta con la base de datos y arranca el modo elegido.
func run() error {
	// Configura slog con salida JSON como logger por defecto
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Lee el modo de despliegue del primer argumento
	if len(os.Args) < 2 {
		return errors.New("missing mode: migrate, api, producer or consumer")
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

	// Arranca el modo elegido (migrate, api, producer o consumer)
	switch mode {
	case "migrate":
		return database.Migrate(ctx, pool, migrations.Payments, "payments")
	case "api":
		return runAPI(ctx, pool)
	case "producer":
		return saga.RunProducer(ctx, pool, brokers, "payments")
	case "consumer":
		return runConsumer(ctx, pool, brokers)
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
}

// runAPI sirve la API HTTP hasta la señal de parada.
func runAPI(ctx context.Context, pool *pgxpool.Pool) error {
	// Crea el repositorio, el servicio (la API no cobra, así que no necesita el importe máximo) y el handler
	repo := payments.NewRepository(pool)
	service := payments.NewService(pool, repo, 0)
	handler := payments.NewHandler(service)

	// Sirve la API hasta la señal de parada
	return httpserver.Run(ctx, apiAddr, payments.Routes(handler))
}

// runConsumer procesa los eventos de los topics de payments hasta la señal de parada.
func runConsumer(ctx context.Context, pool *pgxpool.Pool, brokers []string) error {
	// Lee el importe máximo que se cobra; es obligatorio y tiene que ser un entero mayor que 0
	rawMaxAmount := os.Getenv("MAX_AMOUNT")
	if rawMaxAmount == "" {
		return errors.New("missing MAX_AMOUNT")
	}

	maxAmount, err := strconv.Atoi(rawMaxAmount)
	if err != nil || maxAmount <= 0 {
		return fmt.Errorf("invalid MAX_AMOUNT %q: must be an integer greater than 0", rawMaxAmount)
	}

	// Crea el repositorio y el servicio
	repo := payments.NewRepository(pool)
	service := payments.NewService(pool, repo, maxAmount)

	// Consume los eventos con el grupo payments y los envía a su función del servicio
	return saga.RunConsumer(ctx, brokers, "payments", payments.Topics, payments.EventRoutes(service))
}
