// Productor de Kafka. Lee eventos de la tabla outbox de cada servicio, los publica y elimina, limpiando la tabla
package saga

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

// producerBatchSize es el máximo de filas de la outbox que se publican en cada vuelta.
// producerPollInterval es la pausa tras un lote vacío o fallido.
const (
	producerBatchSize    = 100
	producerPollInterval = 500 * time.Millisecond
)

// outboxRow es una fila de la tabla outbox que contiene los eventos pendientes de publicar
type outboxRow struct {
	id        uuid.UUID
	eventType string
	orderID   string
	payload   []byte
}

// RunProducer Lee todos los eventos que existen en la tabla outbox y los publica en batches.
func RunProducer(ctx context.Context, pool *pgxpool.Pool, brokers []string, topic string) error {
	// Crea el cliente de Kafka (por defecto es idempotente y usa acks=all)
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.DefaultProduceTopic(topic),
	)

	if err != nil {
		return fmt.Errorf("create kafka client: %w", err)
	}

	defer client.Close()

	// Publica lotes hasta la señal de parada; tras un lote vacío o fallido espera antes de reintentar
	for ctx.Err() == nil {
		published, err := publishBatch(ctx, pool, client)
		if err != nil && ctx.Err() == nil {
			slog.Error("publish outbox batch", "error", err)
		}

		if err != nil || published == 0 {
			select {
			case <-ctx.Done():
			case <-time.After(producerPollInterval):
			}
		}
	}

	return nil
}

// publishBatch es un método interno usado por RunProducer para publicar un lote de la outbox en Kafka
// borrar lo publicado, todo en una transacción.
func publishBatch(ctx context.Context, pool *pgxpool.Pool, client *kgo.Client) (int, error) {
	var published int
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		// Prepara la query, con FOR UPDATE SKIP LOCKED omite las que tenga bloqueadas otro productor
		rows, err := tx.Query(ctx, `
			SELECT id, event_type, payload->>'order_id', payload
			FROM outbox
			ORDER BY created_at, id
			LIMIT $1
			FOR UPDATE SKIP LOCKED`, producerBatchSize)

		if err != nil {
			return fmt.Errorf("select outbox: %w", err)
		}

		// Ejecuta el batch de queries
		batch, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (outboxRow, error) {
			var r outboxRow
			err := row.Scan(&r.id, &r.eventType, &r.orderID, &r.payload)
			return r, err
		})

		if err != nil {
			return fmt.Errorf("scan outbox: %w", err)
		}

		if len(batch) == 0 {
			return nil
		}

		// Publica el lote con el order_id como clave y el payload tal cual como valor
		records := make([]*kgo.Record, len(batch))
		ids := make([]uuid.UUID, len(batch))

		for i, r := range batch {
			records[i] = &kgo.Record{Key: []byte(r.orderID), Value: r.payload}
			ids[i] = r.id
		}

		if err := client.ProduceSync(ctx, records...).FirstErr(); err != nil {
			return fmt.Errorf("produce outbox: %w", err)
		}

		// Imprime todo lo publicado
		for _, r := range batch {
			slog.Info("event published", "event_type", r.eventType, "order_id", r.orderID)
		}

		// Borra las filas publicadas de la tabla outbox; si algo falla antes, la transacción se deshace y se reintentan
		if _, err := tx.Exec(ctx, `DELETE FROM outbox WHERE id = ANY($1)`, ids); err != nil {
			return fmt.Errorf("delete outbox: %w", err)
		}

		// Devuelve la cantidad de eventos publicados
		published = len(batch)
		return nil
	})

	return published, err
}
