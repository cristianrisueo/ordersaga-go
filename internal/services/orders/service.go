// Reglas de negocio de orders: crear un pedido junto a su evento y leer un pedido

package orders

import (
	"context"
	"fmt"

	"github.com/cristianrisueo/ordersaga-go/internal/saga"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service aplica las reglas de negocio de orders y decide el límite de cada transacción.
type Service struct {
	pool *pgxpool.Pool
	repo *Repository
}

// NewService crea el servicio con el pool, para abrir transacciones, y el repositorio.
func NewService(pool *pgxpool.Pool, repo *Repository) *Service {
	// Devuelve el servicio listo para usar
	return &Service{pool: pool, repo: repo}
}

// CreateOrder crea un pedido en pending y guarda su evento order.created en la misma transacción.
func (s *Service) CreateOrder(ctx context.Context, in CreateOrderInput) (Order, error) {
	// Genera el id del pedido
	id, err := uuid.NewV7()
	if err != nil {
		return Order{}, fmt.Errorf("generate order id: %w", err)
	}

	// Monta el pedido en pending y su evento order.created con los mismos datos
	order := Order{
		ID:        id,
		ProductID: in.ProductID,
		Quantity:  in.Quantity,
		Amount:    in.Amount,
		Status:    StatusPending,
	}
	event := saga.Event{
		EventType: saga.OrderCreated,
		OrderID:   order.ID,
		ProductID: order.ProductID,
		Quantity:  order.Quantity,
		Amount:    order.Amount,
	}

	// Guarda el pedido y su evento en la misma transacción
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Inserta el pedido
		if err := s.repo.insertOrder(ctx, tx, &order); err != nil {
			return err
		}

		// Inserta su evento en la outbox; si falla, el pedido también se deshace
		return s.repo.insertOutbox(ctx, tx, event)
	})

	// Devuelve el error si lo ha habido o el pedido si todo ha ido bien
	if err != nil {
		return Order{}, fmt.Errorf("create order: %w", err)
	}

	return order, nil
}

// GetOrder devuelve un pedido por su id, o ErrNotFound si no existe.
func (s *Service) GetOrder(ctx context.Context, id uuid.UUID) (Order, error) {
	// Devuelve el pedido leído del repositorio
	return s.repo.getOrder(ctx, id)
}
