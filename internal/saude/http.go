// internal/saude/http.go
//
// Endpoint de saúde (/healthz). Não requer autenticação.

package saude

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ----- Constantes -----

const (
	ROTA_HEALTHZ      = "/healthz"
	STATUS_OK         = "ok"
	CONTENT_TYPE_JSON = "application/json; charset=utf-8"
)

// ----- API pública -----

// Rotas registra /healthz no router.
func Rotas(r chi.Router) {
	r.Get(ROTA_HEALTHZ, handleHealthzHandler)
}

// ----- Handlers (sempre no final) -----

func handleHealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", CONTENT_TYPE_JSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": STATUS_OK})
}
