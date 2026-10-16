// Puerta HTTP de payments: valida la petición, llama al servicio y responde JSON

package payments

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/cristianrisueo/ordersaga-go/internal/platform/httpserver"
	"github.com/google/uuid"
)

// Handler atiende las peticiones HTTP de payments.
type Handler struct {
	service *Service
}

// NewHandler crea el handler sobre el servicio.
func NewHandler(service *Service) *Handler {
	// Devuelve el handler listo para usar
	return &Handler{service: service}
}

// health responde 200 mientras la API esté viva.
func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	// Responde que la API está viva
	httpserver.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// getPayment responde 200 con el pago del pedido, 404 si no existe o 422 si el id no es un UUID.
func (h *Handler) getPayment(w http.ResponseWriter, r *http.Request) {
	// Lee el id del pedido de la ruta; si no es un UUID es un 422
	orderID, err := uuid.Parse(r.PathValue("order_id"))
	if err != nil {
		httpserver.WriteError(w, http.StatusUnprocessableEntity, "El order_id no es un UUID válido")
		return
	}

	// Busca el pago; si no existe es un 404
	payment, err := h.service.GetPayment(r.Context(), orderID)
	if errors.Is(err, ErrNotFound) {
		httpserver.WriteError(w, http.StatusNotFound, "Pago no encontrado")
		return
	}

	if err != nil {
		slog.Error("get payment", "order_id", orderID, "error", err)
		httpserver.WriteError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	// Responde 200 con el pago
	httpserver.WriteJSON(w, http.StatusOK, payment)
}
