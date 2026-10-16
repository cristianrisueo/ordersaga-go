// Reglas de negocio de payments: leer un pago y cobrar o rechazar un pedido junto a su evento

package payments

import (
	"context"
	"fmt"

	"github.com/cristianrisueo/ordersaga-go/internal/saga"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service aplica las reglas de negocio de payments y decide el límite de cada transacción.
type Service struct {
	pool      *pgxpool.Pool
	repo      *Repository
	maxAmount int
}

// NewService crea el servicio con el pool, para abrir transacciones, el repositorio y el importe máximo que se cobra.
func NewService(pool *pgxpool.Pool, repo *Repository, maxAmount int) *Service {
	// Devuelve el servicio listo para usar
	return &Service{pool: pool, repo: repo, maxAmount: maxAmount}
}

// GetPayment devuelve el pago de un pedido, o ErrNotFound si no existe.
func (s *Service) GetPayment(ctx context.Context, orderID uuid.UUID) (Payment, error) {
	// Devuelve el pago leído del repositorio
	return s.repo.getPayment(ctx, orderID)
}

// ChargePayment cobra el pedido, o lo rechaza si supera el importe máximo, y escribe su evento en la misma transacción.
func (s *Service) ChargePayment(ctx context.Context, event saga.Event) error {
	// Decide el estado: completed si el importe no supera el máximo y failed si lo supera
	status := StatusCompleted
	if event.Amount > s.maxAmount {
		status = StatusFailed
	}

	// Guarda el pago y su evento en la outbox en la misma transacción
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Crea el pago a partir de los datos del evento recibido
		payment := Payment{
			OrderID: event.OrderID,
			Amount:  event.Amount,
			Status:  status,
		}

		// Inserta el pago; si ya existía, el evento es repetido y no se hace nada más
		inserted, err := s.repo.insertPayment(ctx, tx, payment)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}

		// Crea una copia del evento para cambiarle el tipo
		next := event

		// Según el estado cambia el tipo a completed o failed e inserta el nuevo evento en la outbox
		next.EventType = saga.PaymentCompleted
		if status == StatusFailed {
			next.EventType = saga.PaymentFailed
		}

		return s.repo.insertOutbox(ctx, tx, next)
	})

	// Devuelve el error si lo ha habido
	if err != nil {
		return fmt.Errorf("charge payment: %w", err)
	}

	// Devuelve nil con el pago y su evento guardados
	return nil
}
