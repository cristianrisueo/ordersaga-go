// Tipos del dominio de stock: el producto, la reserva de un pedido, sus estados y sus errores

package stock

import (
	"errors"

	"github.com/google/uuid"
)

// Estados de una reserva.
const (
	StatusReserved = "reserved" // StatusReserved es el estado de una reserva con las unidades descontadas.
	StatusFailed   = "failed"   // StatusFailed es el estado de una reserva sin unidades suficientes.
	StatusReleased = "released" // StatusReleased es el estado de una reserva cuyas unidades se han devuelto.
)

// ErrNotFound indica que el producto no existe.
var ErrNotFound = errors.New("product not found")

// Product es un producto del catálogo tal y como lo devuelve la API.
type Product struct {
	ID        string `json:"product_id"` // ID identifica el producto.
	Available int    `json:"available"`  // Available es el número de unidades disponibles.
}

// Reservation es la reserva de unidades de un pedido; hay una por pedido.
type Reservation struct {
	OrderID   uuid.UUID // OrderID identifica el pedido de la reserva.
	ProductID string    // ProductID es el producto reservado.
	Quantity  int       // Quantity es el número de unidades reservadas.
	Status    string    // Status es uno de los estados de la reserva.
}
