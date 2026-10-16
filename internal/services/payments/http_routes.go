// Rutas HTTP de payments: qué método y ruta atiende cada función del handler

package payments

import "net/http"

// Routes registra las rutas de payments y devuelve el handler HTTP que las sirve.
func Routes(h *Handler) http.Handler {
	// Crea el Mux
	mux := http.NewServeMux()

	// Registra cada ruta con su método
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /payments/{order_id}", h.getPayment)

	// Devuelve el mux con las rutas registradas
	return mux
}
