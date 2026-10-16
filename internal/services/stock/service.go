// Reglas de negocio de stock: leer un producto, y reservar o liberar las unidades de un pedido junto a su evento

package stock

import (
	"context"
	"fmt"

	"github.com/cristianrisueo/ordersaga-go/internal/saga"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service aplica las reglas de negocio de stock y decide el límite de cada transacción.
type Service struct {
	pool *pgxpool.Pool
	repo *Repository
}

// NewService crea el servicio con el pool, para abrir transacciones, y el repositorio.
func NewService(pool *pgxpool.Pool, repo *Repository) *Service {
	// Devuelve el servicio listo para usar
	return &Service{pool: pool, repo: repo}
}

// GetProduct devuelve un producto por su id, o ErrNotFound si no existe.
func (s *Service) GetProduct(ctx context.Context, id string) (Product, error) {
	// Devuelve el producto leído del repositorio
	return s.repo.getProduct(ctx, id)
}

// ReserveStock reserva las unidades del pedido y escribe stock.reserved o stock.failed en la misma transacción.
func (s *Service) ReserveStock(ctx context.Context, event saga.Event) error {
	// Reserva las unidades y guarda el evento en outbox en la misma transacción
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Crea la reserva a partir de los datos del evento recibido
		reservation := Reservation{
			OrderID:   event.OrderID,
			ProductID: event.ProductID,
			Quantity:  event.Quantity,
			Status:    StatusReserved,
		}

		// Inserta la reserva; si ya existía, el evento es repetido y no se hace nada más
		inserted, err := s.repo.insertReservation(ctx, tx, reservation)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}

		// Descuenta las unidades solo si hay suficiente stock (si el producto no existe, tampoco descuenta nada)
		hasUnits, err := s.repo.reserveUnits(ctx, tx, event.ProductID, event.Quantity)
		if err != nil {
			return err
		}

		// Sin unidades suficientes, la reserva pasa a failed; la fila no se borra por idempotencia
		if !hasUnits {
			if err := s.repo.markReservationFailed(ctx, tx, event.OrderID); err != nil {
				return err
			}
		}

		// Crea una copia del evento para cambiarle el tipo
		next := event

		// Dependiendo del resultado cambia el tipo a reserved o failed e inserta el nuevo evento en outbox
		next.EventType = saga.StockReserved
		if !hasUnits {
			next.EventType = saga.StockFailed
		}

		return s.repo.insertOutbox(ctx, tx, next)
	})

	// Devuelve el error si lo ha habido
	if err != nil {
		return fmt.Errorf("reserve stock: %w", err)
	}

	// Devuelve nil con la reserva y su evento guardados
	return nil
}

// ReleaseStock libera las unidades reservadas del pedido y escribe stock.released en la misma transacción.
func (s *Service) ReleaseStock(ctx context.Context, event saga.Event) error {
	// Libera la reserva, devuelve las unidades y guarda el evento en outbox en la misma transacción
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Libera la reserva; si no había nada que liberar (evento repetido o sin reserva) no se hace nada más
		reservation, released, err := s.repo.releaseReservation(ctx, tx, event.OrderID)
		if err != nil {
			return err
		}
		if !released {
			return nil
		}

		// Devuelve al producto las unidades de la reserva, nunca las del evento
		if err := s.repo.returnUnits(ctx, tx, reservation.ProductID, reservation.Quantity); err != nil {
			return err
		}

		// Crea una copia del evento con el tipo stock.released e inserta el nuevo evento en outbox
		next := event
		next.EventType = saga.StockReleased

		return s.repo.insertOutbox(ctx, tx, next)
	})

	// Devuelve el error si lo ha habido
	if err != nil {
		return fmt.Errorf("release stock: %w", err)
	}

	// Devuelve nil con la reserva liberada y su evento guardado
	return nil
}
