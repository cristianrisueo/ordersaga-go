// Acceso a la base de datos de payments: los pagos y la outbox

package payments

import (
	"context"
	"errors"
	"fmt"

	"github.com/cristianrisueo/ordersaga-go/internal/saga"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository ejecuta el SQL de payments.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository crea el repositorio sobre el pool de conexiones.
func NewRepository(pool *pgxpool.Pool) *Repository {
	// Devuelve el repositorio listo para usar
	return &Repository{pool: pool}
}

// getPayment lee el pago de un pedido y devuelve ErrNotFound si no existe.
func (r *Repository) getPayment(ctx context.Context, orderID uuid.UUID) (Payment, error) {
	// Lee el pago
	var payment Payment
	err := r.pool.QueryRow(ctx, `
		SELECT order_id, amount, status
		FROM payments
		WHERE order_id = $1`, orderID,
	).Scan(&payment.OrderID, &payment.Amount, &payment.Status)

	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, ErrNotFound
	}

	if err != nil {
		return Payment{}, fmt.Errorf("get payment: %w", err)
	}

	// Devuelve el pago leído
	return payment, nil
}

// insertPayment inserta el pago y devuelve false si ya existía, es decir, si el evento es repetido.
func (r *Repository) insertPayment(ctx context.Context, tx pgx.Tx, payment Payment) (bool, error) {
	// Inserta el pago; si ya existía no afecta a ninguna fila
	tag, err := tx.Exec(ctx, `
		INSERT INTO payments (order_id, amount, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (order_id) DO NOTHING`,
		payment.OrderID, payment.Amount, payment.Status,
	)

	if err != nil {
		return false, fmt.Errorf("insert payment: %w", err)
	}

	// Devuelve true si el pago es nuevo o false si el evento es repetido
	return tag.RowsAffected() == 1, nil
}

// insertOutbox guarda el evento en la tabla outbox para que lo publique el productor.
func (r *Repository) insertOutbox(ctx context.Context, tx pgx.Tx, event saga.Event) error {
	// Inserta el tipo y el evento entero como payload
	_, err := tx.Exec(ctx, `INSERT INTO outbox (event_type, payload) VALUES ($1, $2)`, event.EventType, event)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	// Devuelve nil si todo ha ido correctamente
	return nil
}
