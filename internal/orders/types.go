// Tipos del dominio de orders: el pedido, la entrada para crearlo, sus estados y sus errores

package orders

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Estados de un pedido.
const (
	StatusPending   = "pending"   // StatusPending es el estado de un pedido recién creado.
	StatusConfirmed = "confirmed" // StatusConfirmed es el estado de un pedido reservado y cobrado.
	StatusCancelled = "cancelled" // StatusCancelled es el estado de un pedido sin stock o sin cobro.
)

// ErrNotFound indica que el pedido no existe.
var ErrNotFound = errors.New("order not found")

// Order es un pedido tal y como lo guarda la base de datos y lo devuelve la API.
type Order struct {
	ID        uuid.UUID `json:"id"`         // ID identifica el pedido.
	ProductID string    `json:"product_id"` // ProductID es el producto pedido.
	Quantity  int       `json:"quantity"`   // Quantity es el número de unidades pedidas.
	Amount    int       `json:"amount"`     // Amount es el importe del pedido.
	Status    string    `json:"status"`     // Status es uno de los estados del pedido.
	CreatedAt time.Time `json:"created_at"` // CreatedAt es el momento en que se guardó el pedido.
}

// CreateOrderInput es el cuerpo del POST /orders.
type CreateOrderInput struct {
	ProductID string `json:"product_id"` // ProductID es el producto pedido.
	Quantity  int    `json:"quantity"`   // Quantity es el número de unidades pedidas.
	Amount    int    `json:"amount"`     // Amount es el importe del pedido.
}
