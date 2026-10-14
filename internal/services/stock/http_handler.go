// Puerta HTTP de stock: llama al servicio y responde JSON

package stock

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Handler atiende las peticiones HTTP de stock.
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

// getStock responde 200 con las unidades disponibles del producto, o 404 si no existe.
func (h *Handler) getStock(w http.ResponseWriter, r *http.Request) {
	// Busca el producto; si no existe es un 404
	productID := r.PathValue("product_id")
	product, err := h.service.GetProduct(r.Context(), productID)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "Producto no encontrado")
		return
	}

	if err != nil {
		slog.Error("get product", "product_id", productID, "error", err)
		writeError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	// Responde 200 con el producto
	writeJSON(w, http.StatusOK, product)
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
