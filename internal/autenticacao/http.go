// internal/autenticacao/http.go
//
// Handler HTTP do endpoint público POST /login. Aceita {"senha": "..."}
// e devolve {"token": "...", "expira_em": "RFC3339"}. Senha errada → 401.

package autenticacao

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"SignalHub/internal/resposta"
)

// ----- Constantes -----

const ROTA_LOGIN = "/login"

// ----- Tipo Handler -----

// Handler expõe o endpoint /login usando um Servico injetado.
type Handler struct {
	servico *Servico
}

// NovoHandler constrói um Handler.
func NovoHandler(servico *Servico) *Handler {
	return &Handler{servico: servico}
}

// Rotas registra POST /login (rota pública).
func (h *Handler) Rotas(r chi.Router) {
	r.Post(ROTA_LOGIN, h.login)
}

// ----- Endpoints (handlers no final) -----

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var entrada EntradaLogin
	if err := resposta.LerJSON(r, &entrada); err != nil {
		resposta.Erro(w, http.StatusBadRequest, "corpo JSON inválido")
		return
	}
	if entrada.Senha == "" {
		resposta.Erro(w, http.StatusBadRequest, "senha é obrigatória")
		return
	}

	token, expiraEm, err := h.servico.Autenticar(entrada.Senha)
	if errors.Is(err, ErroNaoAutorizado) {
		resposta.Erro(w, http.StatusUnauthorized, "senha inválida")
		return
	}
	if err != nil {
		resposta.Erro(w, http.StatusInternalServerError, err.Error())
		return
	}

	resposta.JSON(w, http.StatusOK, RespostaLogin{
		Token:    token,
		ExpiraEm: expiraEm.Format(time.RFC3339),
	})
}
