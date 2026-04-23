// internal/saude/http.go
//
// Endpoint de saúde (/healthz). Não requer autenticação.

package saude

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Rotas registra /healthz no router.
func Rotas(r chi.Router) {
	r.Get("/healthz", handleHealthz)
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
