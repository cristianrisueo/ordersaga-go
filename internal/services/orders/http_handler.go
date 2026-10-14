// Puerta HTTP de orders: valida la petición, llama al servicio y responde JSON

package orders

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// Handler atiende las peticiones HTTP de orders.
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
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// createOrder crea un pedido y responde 201 con él, o 422 si el cuerpo no es válido.
func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	// Decodifica el cuerpo; un JSON mal formado o con tipos incorrectos es un 422
	var in CreateOrderInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "El cuerpo no es un JSON válido")
		return
	}

	// Valida los campos del pedido
	if detail := validateCreateOrder(in); detail != "" {
		writeError(w, http.StatusUnprocessableEntity, detail)
		return
	}

	// Crea el pedido
	order, err := h.service.CreateOrder(r.Context(), in)
	if err != nil {
		slog.Error("create order", "error", err)
		writeError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	// Responde 201 con el pedido creado
	writeJSON(w, http.StatusCreated, order)
}

// getOrder responde 200 con el pedido, 404 si no existe o 422 si el id no es un UUID.
func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	// Lee el id de la ruta; si no es un UUID es un 422
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "El id no es un UUID válido")
		return
	}

	// Busca el pedido; si no existe es un 404
	order, err := h.service.GetOrder(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "Pedido no encontrado")
		return
	}

	if err != nil {
		slog.Error("get order", "order_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	// Responde 200 con el pedido
	writeJSON(w, http.StatusOK, order)
}

// validateCreateOrder devuelve el motivo por el que el pedido no es válido, o "" si todo está bien.
func validateCreateOrder(in CreateOrderInput) string {
	// Comprueba cada campo y devuelve el primer motivo de rechazo
	switch {
	case in.ProductID == "":
		return "product_id no puede estar vacío"
	case in.Quantity <= 0:
		return "quantity debe ser mayor que 0"
	case in.Amount <= 0:
		return "amount debe ser mayor que 0"
	default:
		return ""
	}
}

// writeJSON responde con el código de estado y el valor codificado en JSON.
func writeJSON(w http.ResponseWriter, status int, value any) {
	// Escribe la cabecera y el código de estado
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Codifica el valor; si falla, la respuesta ya ha empezado y solo se puede registrar
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("write response", "error", err)
	}
}

// writeError responde con el código de estado y el error en el formato de FastAPI.
func writeError(w http.ResponseWriter, status int, detail string) {
	// Responde {"detail": "..."}
	writeJSON(w, status, map[string]string{"detail": detail})
}
