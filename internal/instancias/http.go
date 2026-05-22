// internal/instancias/http.go
//
// Handlers HTTP do CRUD de instâncias Zabbix e MSP Clouds + registro de
// rotas Chi. GET /zabbix/instancias/{id} devolve os filtros aninhados.

package instancias

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"SignalHub/internal/resposta"
)

// ----- Constantes -----

const (
	ROTA_ZABBIX_COLECAO = "/zabbix/instancias"
	ROTA_ZABBIX_ITEM    = "/zabbix/instancias/{id}"
	ROTA_MSP_COLECAO    = "/mspclouds/instancias"
	ROTA_MSP_ITEM       = "/mspclouds/instancias/{id}"
	PARAM_ID            = "id"
)

// ----- Tipo Handler -----

// Handler expõe os endpoints de instâncias usando um Servico injetado.
type Handler struct {
	servico *Servico
}

// NovoHandler constrói um Handler.
func NovoHandler(servico *Servico) *Handler {
	return &Handler{servico: servico}
}

// Rotas registra o CRUD de instâncias Zabbix e MSP Clouds no router.
func (h *Handler) Rotas(r chi.Router) {
	r.Get(ROTA_ZABBIX_COLECAO, h.listarZabbix)
	r.Post(ROTA_ZABBIX_COLECAO, h.criarZabbix)
	r.Get(ROTA_ZABBIX_ITEM, h.buscarZabbix)
	r.Put(ROTA_ZABBIX_ITEM, h.atualizarZabbix)
	r.Delete(ROTA_ZABBIX_ITEM, h.removerZabbix)

	r.Get(ROTA_MSP_COLECAO, h.listarMsp)
	r.Post(ROTA_MSP_COLECAO, h.criarMsp)
	r.Get(ROTA_MSP_ITEM, h.buscarMsp)
	r.Put(ROTA_MSP_ITEM, h.atualizarMsp)
	r.Delete(ROTA_MSP_ITEM, h.removerMsp)
}

// ----- Endpoints Zabbix (handlers no final) -----

func (h *Handler) listarZabbix(w http.ResponseWriter, r *http.Request) {
	lista, err := h.servico.ListarZabbix(r.Context())
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, map[string]any{"data": lista})
}

func (h *Handler) buscarZabbix(w http.ResponseWriter, r *http.Request) {
	id, err := resposta.IDdaRota(r, PARAM_ID)
	if err != nil {
		resposta.Erro(w, http.StatusBadRequest, "id inválido")
		return
	}

	instancia, err := h.servico.BuscarZabbixComFiltros(r.Context(), id)
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, instancia)
}

func (h *Handler) criarZabbix(w http.ResponseWriter, r *http.Request) {
	var entrada EntradaZabbix
	if err := resposta.LerJSON(r, &entrada); err != nil {
		resposta.Erro(w, http.StatusBadRequest, "corpo JSON inválido")
		return
	}

	instancia, err := h.servico.CriarZabbix(r.Context(), entrada)
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusCreated, instancia)
}

func (h *Handler) atualizarZabbix(w http.ResponseWriter, r *http.Request) {
	id, err := resposta.IDdaRota(r, PARAM_ID)
	if err != nil {
		resposta.Erro(w, http.StatusBadRequest, "id inválido")
		return
	}

	var entrada EntradaZabbix
	if err := resposta.LerJSON(r, &entrada); err != nil {
		resposta.Erro(w, http.StatusBadRequest, "corpo JSON inválido")
		return
	}

	instancia, err := h.servico.AtualizarZabbix(r.Context(), id, entrada)
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, instancia)
}

func (h *Handler) removerZabbix(w http.ResponseWriter, r *http.Request) {
	id, err := resposta.IDdaRota(r, PARAM_ID)
	if err != nil {
		resposta.Erro(w, http.StatusBadRequest, "id inválido")
		return
	}

	if err := h.servico.RemoverZabbix(r.Context(), id); err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, map[string]bool{"removido": true})
}

// ----- Endpoints MSP Clouds (handlers no final) -----

func (h *Handler) listarMsp(w http.ResponseWriter, r *http.Request) {
	lista, err := h.servico.ListarMsp(r.Context())
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, map[string]any{"data": lista})
}

func (h *Handler) buscarMsp(w http.ResponseWriter, r *http.Request) {
	id, err := resposta.IDdaRota(r, PARAM_ID)
	if err != nil {
		resposta.Erro(w, http.StatusBadRequest, "id inválido")
		return
	}

	instancia, err := h.servico.BuscarMsp(r.Context(), id)
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, instancia)
}

func (h *Handler) criarMsp(w http.ResponseWriter, r *http.Request) {
	var entrada EntradaMsp
	if err := resposta.LerJSON(r, &entrada); err != nil {
		resposta.Erro(w, http.StatusBadRequest, "corpo JSON inválido")
		return
	}

	instancia, err := h.servico.CriarMsp(r.Context(), entrada)
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusCreated, instancia)
}

func (h *Handler) atualizarMsp(w http.ResponseWriter, r *http.Request) {
	id, err := resposta.IDdaRota(r, PARAM_ID)
	if err != nil {
		resposta.Erro(w, http.StatusBadRequest, "id inválido")
		return
	}

	var entrada EntradaMsp
	if err := resposta.LerJSON(r, &entrada); err != nil {
		resposta.Erro(w, http.StatusBadRequest, "corpo JSON inválido")
		return
	}

	instancia, err := h.servico.AtualizarMsp(r.Context(), id, entrada)
	if err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, instancia)
}

func (h *Handler) removerMsp(w http.ResponseWriter, r *http.Request) {
	id, err := resposta.IDdaRota(r, PARAM_ID)
	if err != nil {
		resposta.Erro(w, http.StatusBadRequest, "id inválido")
		return
	}

	if err := h.servico.RemoverMsp(r.Context(), id); err != nil {
		resposta.Tratar(w, err)
		return
	}
	resposta.JSON(w, http.StatusOK, map[string]bool{"removido": true})
}
