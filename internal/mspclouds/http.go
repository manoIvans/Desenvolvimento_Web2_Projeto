// internal/mspclouds/http.go
//
// Handlers HTTP do domínio MSP Clouds + registro de rotas Chi.

package mspclouds

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler expõe os endpoints MSP Clouds usando um Servico injetado.
type Handler struct {
	servico *Servico
}

// NovoHandler constrói um Handler.
func NovoHandler(servico *Servico) *Handler {
	return &Handler{servico: servico}
}

// Rotas registra /mspclouds e /mspclouds/refresh.
func (h *Handler) Rotas(r chi.Router) {
	r.Get("/mspclouds", h.listar)
	r.Post("/mspclouds/refresh", h.refresh)
}

// ----- Endpoints -----

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	alertas, falhas := h.servico.CarregarCache()
	respJSON(w, http.StatusOK, map[string]any{
		"data":   alertas,
		"falhas": falhas,
	})
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	if !h.servico.Configurado() {
		respErro(w, http.StatusBadRequest, "nenhuma api_key MSP Clouds configurada")
		return
	}

	resultado, err := h.servico.AtualizarERetornar()
	if err != nil {
		respErro(w, http.StatusInternalServerError, err.Error())
		return
	}
	respJSON(w, http.StatusOK, resultado)
}

// ----- Helpers HTTP -----

func respJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func respErro(w http.ResponseWriter, status int, msg string) {
	respJSON(w, status, map[string]string{"error": msg})
}
