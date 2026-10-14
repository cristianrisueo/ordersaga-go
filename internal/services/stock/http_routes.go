// Rutas HTTP de stock: qué método y ruta atiende cada función del handler

package stock

import "net/http"

// Routes registra las rutas de stock y devuelve el handler HTTP que las sirve.
func Routes(h *Handler) http.Handler {
	// Crea el Mux
	mux := http.NewServeMux()

	// Registra cada ruta con su método
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /stock/{product_id}", h.getStock)

	// Devuelve el mux con las rutas registradas
	return mux
}
