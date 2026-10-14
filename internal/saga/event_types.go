// Define el evento de la saga y el método para decodificar eventos

package saga

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Tipos de evento de la saga.
const (
	OrderCreated     = "order.created"     // OrderCreated lo publica orders al crear un pedido.
	StockReserved    = "stock.reserved"    // StockReserved lo publica stock al reservar las unidades del pedido.
	StockFailed      = "stock.failed"      // StockFailed lo publica stock cuando no hay unidades suficientes.
	StockReleased    = "stock.released"    // StockReleased lo publica stock al liberar una reserva tras un pago fallido.
	PaymentCompleted = "payment.completed" // PaymentCompleted lo publica payments al cobrar el pedido.
	PaymentFailed    = "payment.failed"    // PaymentFailed lo publica payments cuando el cobro falla.
)

// Event es el evento de la saga; En una saga por coreografía todos los servicios publican y reciben el mismo tipo
type Event struct {
	EventType string    `json:"event_type"` // EventType es uno de los tipos de evento de la saga.
	OrderID   uuid.UUID `json:"order_id"`   // OrderID identifica el pedido y es la clave del mensaje en Kafka.
	ProductID string    `json:"product_id"` // ProductID es el producto del pedido.
	Quantity  int       `json:"quantity"`   // Quantity es el número de unidades pedidas.
	Amount    int       `json:"amount"`     // Amount es el importe del pedido.
}

// DecodeEvent decodifica el JSON de un mensaje y comprueba que trae tipo y order_id.
func DecodeEvent(data []byte) (Event, error) {
	// Decodifica el JSON en event (un order_id que no es un UUID falla aquí)
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return Event{}, fmt.Errorf("decode event: %w", err)
	}

	// Comprueba los campos que necesita cualquier servicio para el mensaje
	if event.EventType == "" {
		return Event{}, errors.New("decode event: missing event_type")
	}
	if event.OrderID == uuid.Nil {
		return Event{}, errors.New("decode event: missing order_id")
	}

	// Devuelve el vento casteado si todo ha ido bien
	return event, nil
}
