// Rutas HTTP de orders: qué método y ruta atiende cada función del handler

package orders

import "net/http"

// Routes registra las rutas de orders y devuelve el handler HTTP que las sirve.
func Routes(h *Handler) http.Handler {
	// Crea el Mux
	mux := http.NewServeMux()

	// Registra cada ruta con su método
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /orders", h.createOrder)
	mux.HandleFunc("GET /orders/{id}", h.getOrder)

	// Devuelve el mux con las rutas registradas
	return mux
}
