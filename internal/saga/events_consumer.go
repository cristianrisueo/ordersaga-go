// Consumidor de eventos de Kafka. Lee todo lo publicado en su topic suscrito y redirige al handler de cada servicio

package saga

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
)

// RunConsumer lee los topics del servicio y confirma cada offset solo después de procesar su mensaje.
func RunConsumer(ctx context.Context, brokers []string, group string, topics []string, handle func(context.Context, Event) error) error {
	// Crea el cliente del grupo con commit manual; un grupo nuevo empieza por el principio de cada topic
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),
		kgo.DisableAutoCommit(),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.BlockRebalanceOnPoll(),
	)

	if err != nil {
		return fmt.Errorf("create kafka client: %w", err)
	}

	defer client.CloseAllowingRebalance()

	// El mensaje en curso termina y se confirma aunque llegue la señal de parada
	workCtx := context.WithoutCancel(ctx)

	for {
		// Espera el siguiente lote; sale sin error con la señal de parada o el cliente cerrado
		fetches := client.PollFetches(ctx)
		if ctx.Err() != nil || fetches.IsClientClosed() {
			return nil
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("fetch %s/%d: %w", errs[0].Topic, errs[0].Partition, errs[0].Err)
		}

		// Procesa los registros en orden y confirma cada uno; si falla algo, el proceso muere sin confirmar más
		for _, record := range fetches.Records() {
			if ctx.Err() != nil {
				break
			}

			event, err := DecodeEvent(record.Value)
			if err != nil {
				return fmt.Errorf("record %s/%d@%d: %w", record.Topic, record.Partition, record.Offset, err)
			}
			if err := handle(workCtx, event); err != nil {
				return fmt.Errorf("handle %s %s: %w", event.EventType, event.OrderID, err)
			}
			if err := client.CommitRecords(workCtx, record); err != nil {
				return fmt.Errorf("commit %s/%d@%d: %w", record.Topic, record.Partition, record.Offset, err)
			}
			slog.Info("event processed", "event_type", event.EventType, "order_id", event.OrderID)
		}

		// Permite el rebalanceo que haya quedado pendiente durante el lote
		client.AllowRebalance()
	}
}
