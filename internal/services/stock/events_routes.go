// Puerta de Kafka de stock: los topics que escucha y a qué función del servicio va cada evento

package stock

import (
	"context"

	"github.com/cristianrisueo/ordersaga-go/internal/saga"
)

// Topics son los topics de Kafka que escucha stock.
var Topics = []string{"orders", "payments"}

// EventRoutes devuelve la función que envía cada evento a su función del servicio e ignora el resto.
func EventRoutes(s *Service) func(context.Context, saga.Event) error {
	return func(ctx context.Context, event saga.Event) error {
		// Envía cada evento a su función del servicio; los que no le interesan se ignoran
		switch event.EventType {
		case saga.OrderCreated:
			return s.ReserveStock(ctx, event)
		case saga.PaymentFailed:
			return s.ReleaseStock(ctx, event)
		default:
			return nil
		}
	}
}
