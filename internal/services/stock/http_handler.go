// Puerta HTTP de stock: llama al servicio y responde JSON

package stock

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/cristianrisueo/ordersaga-go/internal/platform/httpserver"
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
	httpserver.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// getStock responde 200 con las unidades disponibles del producto, o 404 si no existe.
func (h *Handler) getStock(w http.ResponseWriter, r *http.Request) {
	// Obtiene el ID del producto
	productID := r.PathValue("product_id")

	// Busca el producto; si no existe es un 404
	product, err := h.service.GetProduct(r.Context(), productID)
	if errors.Is(err, ErrNotFound) {
		httpserver.WriteError(w, http.StatusNotFound, "Producto no encontrado")
		return
	}

	if err != nil {
		slog.Error("get product", "product_id", productID, "error", err)
		httpserver.WriteError(w, http.StatusInternalServerError, "Error interno del servidor")
		return
	}

	// Responde 200 con el producto
	httpserver.WriteJSON(w, http.StatusOK, product)
}
