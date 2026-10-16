// Tipos del dominio de payments: el pago de un pedido, sus estados y sus errores

package payments

import (
	"errors"

	"github.com/google/uuid"
)

// Estados de un pago.
const (
	StatusCompleted = "completed" // StatusCompleted es el estado de un pago cobrado.
	StatusFailed    = "failed"    // StatusFailed es el estado de un pago rechazado por superar el importe máximo.
)

// ErrNotFound indica que el pago no existe.
var ErrNotFound = errors.New("payment not found")

// Payment es el pago de un pedido tal y como lo devuelve la API; hay uno por pedido.
type Payment struct {
	OrderID uuid.UUID `json:"order_id"` // OrderID identifica el pedido del pago.
	Amount  int       `json:"amount"`   // Amount es el importe del pago.
	Status  string    `json:"status"`   // Status es uno de los estados del pago.
}
