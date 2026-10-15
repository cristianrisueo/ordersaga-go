// Formato de respuesta común de las APIs: JSON y errores como {"detail": "..."}

package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSON responde con el código de estado y el valor codificado en JSON.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	// Escribe la cabecera y el código de estado
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Codifica el valor; si falla, la respuesta ya ha empezado y solo se puede registrar
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("write response", "error", err)
	}
}

// WriteError responde con el código de estado y el error en el formato de FastAPI.
func WriteError(w http.ResponseWriter, status int, detail string) {
	// Responde {"detail": "..."}
	WriteJSON(w, status, map[string]string{"detail": detail})
}
