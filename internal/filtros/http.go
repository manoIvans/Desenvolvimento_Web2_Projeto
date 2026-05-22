// internal/filtros/http.go
//
// Handlers HTTP do CRUD de filtros + registro de rotas Chi.

package filtros

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"SignalHub/internal/resposta"
)

// ----- Constantes -----

const (
	ROTA_COLECAO = "/filtros"
	ROTA_ITEM    = "/filtros/{id}"
	PARAM_ID     = "id"
)

// ----- Tipo Handler -----

// Handler expõe os endpoints de filtros usando um Servico injetado.
type Handler struct {
	servico *Servico
}

// NovoHandler constrói um Handler.
func NovoHandler(servico *Servico) *Handler {
	return &Handler{servico: servico}
}

// Rotas registra o CRUD de filtros no router.
func (h *Handler) Rotas(r chi.Router) {
	r.Get(ROTA_COLECAO, h.listar)
	r.Post(ROTA_COLECAO, h.criar)
	r.Get(ROTA_ITEM, h.buscar)
	r.Put(ROTA_ITEM, h.atualizar)
	r.Delete(ROTA_ITEM, h.remover)
}

// ----- Endpoints (handlers no final) -----

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) {
	lista, err := h.servico.Listar(r.Context())
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, map[string]any{"data": lista})
}

func (h *Handler) buscar(w http.ResponseWriter, r *http.Request) {
	id, err := resposta.IDdaRota(r, PARAM_ID)
	if err != nil {
		resposta.Erro(w, http.StatusBadRequest, "id inválido")
		return
	}

	filtro, err := h.servico.Buscar(r.Context(), id)
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, filtro)
}

func (h *Handler) criar(w http.ResponseWriter, r *http.Request) {
	var entrada EntradaFiltro
	if err := resposta.LerJSON(r, &entrada); err != nil {
		resposta.Erro(w, http.StatusBadRequest, "corpo JSON inválido")
		return
	}

	filtro, err := h.servico.Criar(r.Context(), entrada)
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusCreated, filtro)
}

func (h *Handler) atualizar(w http.ResponseWriter, r *http.Request) {
	id, err := resposta.IDdaRota(r, PARAM_ID)
	if err != nil {
		resposta.Erro(w, http.StatusBadRequest, "id inválido")
		return
	}

	var entrada EntradaFiltro
	if err := resposta.LerJSON(r, &entrada); err != nil {
		resposta.Erro(w, http.StatusBadRequest, "corpo JSON inválido")
		return
	}

	filtro, err := h.servico.Atualizar(r.Context(), id, entrada)
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, filtro)
}

func (h *Handler) remover(w http.ResponseWriter, r *http.Request) {
	id, err := resposta.IDdaRota(r, PARAM_ID)
	if err != nil {
		resposta.Erro(w, http.StatusBadRequest, "id inválido")
		return
	}

	if err := h.servico.Remover(r.Context(), id); err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, map[string]bool{"removido": true})
}
