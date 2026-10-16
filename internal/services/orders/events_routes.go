// Puerta de Kafka de orders: los topics que escucha y a qué función del servicio va cada evento

package orders

import (
	"context"

	"github.com/cristianrisueo/ordersaga-go/internal/saga"
)

// Topics son los topics de Kafka que escucha orders.
var Topics = []string{"stock", "payments"}

// EventRoutes devuelve la función que envía cada evento a su función del servicio e ignora el resto.
func EventRoutes(s *Service) func(context.Context, saga.Event) error {
	return func(ctx context.Context, event saga.Event) error {
		// Cancela solo cuando stock ha terminado (stock.failed o stock.released): payment.failed y el resto se ignoran
		switch event.EventType {
		case saga.PaymentCompleted:
			return s.ConfirmOrder(ctx, event)
		case saga.StockFailed, saga.StockReleased:
			return s.CancelOrder(ctx, event)
		default:
			return nil
		}
	}
}
