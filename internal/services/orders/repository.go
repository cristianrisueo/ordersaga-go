// Acceso a la base de datos de orders: los pedidos y la outbox

package orders

import (
	"context"
	"errors"
	"fmt"

	"github.com/cristianrisueo/ordersaga-go/internal/saga"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository ejecuta el SQL de orders.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository crea el repositorio sobre el pool de conexiones.
func NewRepository(pool *pgxpool.Pool) *Repository {
	// Devuelve el repositorio listo para usar
	return &Repository{pool: pool}
}

// insertOrder inserta el pedido y rellena su created_at con el que pone la base de datos.
func (r *Repository) insertOrder(ctx context.Context, tx pgx.Tx, order *Order) error {
	// Inserta el pedido y lee su created_at
	err := tx.QueryRow(ctx, `
		INSERT INTO orders (id, product_id, quantity, amount, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at`,
		order.ID, order.ProductID, order.Quantity, order.Amount, order.Status,
	).Scan(&order.CreatedAt)

	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	// Devuelve nil con el pedido insertado
	return nil
}

// getOrder lee un pedido por su id y devuelve ErrNotFound si no existe.
func (r *Repository) getOrder(ctx context.Context, id uuid.UUID) (Order, error) {
	// Lee el pedido
	var order Order
	err := r.pool.QueryRow(ctx, `
		SELECT id, product_id, quantity, amount, status, created_at
		FROM orders
		WHERE id = $1`, id,
	).Scan(&order.ID, &order.ProductID, &order.Quantity, &order.Amount, &order.Status, &order.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}

	if err != nil {
		return Order{}, fmt.Errorf("get order: %w", err)
	}

	// Devuelve el pedido leído
	return order, nil
}

// updateStatusIfPending cambia el estado del pedido solo si sigue en pending; un evento repetido no toca ninguna fila.
func (r *Repository) updateStatusIfPending(ctx context.Context, id uuid.UUID, status string) error {
	// Cambia el estado solo si el pedido sigue en pending
	_, err := r.pool.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1 AND status = 'pending'`, id, status)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	// Devuelve nil tanto si ha cambiado el estado como si el pedido ya no estaba en pending
	return nil
}

// insertOutbox guarda el evento en la outbox para que lo publique el productor.
func (r *Repository) insertOutbox(ctx context.Context, tx pgx.Tx, event saga.Event) error {
	// Inserta el tipo y el evento entero como payload (pgx lo serializa a JSON en la columna JSONB)
	_, err := tx.Exec(ctx, `INSERT INTO outbox (event_type, payload) VALUES ($1, $2)`, event.EventType, event)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	// Devuelve nil con el evento guardado
	return nil
}
