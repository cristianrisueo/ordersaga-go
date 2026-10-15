// Acceso a la base de datos de stock: los productos, las reservas y la outbox

package stock

import (
	"context"
	"errors"
	"fmt"

	"github.com/cristianrisueo/ordersaga-go/internal/saga"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository ejecuta el SQL de stock.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository crea el repositorio sobre el pool de conexiones.
func NewRepository(pool *pgxpool.Pool) *Repository {
	// Devuelve el repositorio listo para usar
	return &Repository{pool: pool}
}

// getProduct lee un producto por su id y devuelve ErrNotFound si no existe.
func (r *Repository) getProduct(ctx context.Context, id string) (Product, error) {
	// Lee el producto
	var product Product

	err := r.pool.QueryRow(ctx, `
		SELECT id, available
		FROM products
		WHERE id = $1`, id,
	).Scan(&product.ID, &product.Available)

	// Devuelve errores en caso de que el producto no exista o se de otro
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}

	if err != nil {
		return Product{}, fmt.Errorf("get product: %w", err)
	}

	// Devuelve el producto leído
	return product, nil
}

// insertReservation inserta la reserva y devuelve false si ya existía, es decir, si el evento es repetido.
func (r *Repository) insertReservation(ctx context.Context, tx pgx.Tx, reservation Reservation) (bool, error) {
	// Inserta la reserva; si ya existía no afecta a ninguna fila
	tag, err := tx.Exec(ctx, `
		INSERT INTO reservations (order_id, product_id, quantity, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (order_id) DO NOTHING`,
		reservation.OrderID, reservation.ProductID, reservation.Quantity, reservation.Status,
	)

	// Devuelve error si se produce
	if err != nil {
		return false, fmt.Errorf("insert reservation: %w", err)
	}

	// Devuelve true si la reserva es nueva o false si el evento es repetido
	return tag.RowsAffected() == 1, nil
}

// reserveUnits descuenta las unidades y devuelve false si no hay suficientes o el producto no existe.
func (r *Repository) reserveUnits(ctx context.Context, tx pgx.Tx, productID string, quantity int) (bool, error) {
	// Realiza el update
	tag, err := tx.Exec(ctx, `
		UPDATE products SET available = available - $2
		WHERE id = $1 AND available >= $2`,
		productID, quantity,
	)

	// Devuelve error si existe
	if err != nil {
		return false, fmt.Errorf("reserve units: %w", err)
	}

	// Devuelve true si la ejecución ha afectado a una fila
	return tag.RowsAffected() == 1, nil
}

// markReservationFailed pasa la reserva del pedido a failed.
func (r *Repository) markReservationFailed(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) error {
	// Cambia el estado de la reserva
	_, err := tx.Exec(ctx, `UPDATE reservations SET status = $2 WHERE order_id = $1`, orderID, StatusFailed)
	if err != nil {
		return fmt.Errorf("mark reservation failed: %w", err)
	}

	// Devuelve nil si la reserva se ha cambiado correctamente
	return nil
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
