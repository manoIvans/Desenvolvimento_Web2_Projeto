// internal/autenticacao/http.go
//
// Handlers HTTP públicos do domínio Autenticação:
//   - POST /login   {"senha"}          → par de tokens
//   - POST /refresh {"refresh_token"}  → novo par (rotaciona o refresh)
//   - POST /logout  {"refresh_token"}  → revoga o refresh token
//
// Senha errada ou refresh inválido/expirado resultam em 401.

package autenticacao

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"SignalHub/internal/resposta"
)

// ----- Constantes -----

const (
	ROTA_LOGIN   = "/login"
	ROTA_REFRESH = "/refresh"
	ROTA_LOGOUT  = "/logout"
	TIPO_BEARER  = "Bearer"
)

// ----- Tipo Handler -----

// Handler expõe os endpoints de autenticação usando um Servico injetado.
type Handler struct {
	servico *Servico
}

// NovoHandler constrói um Handler.
func NovoHandler(servico *Servico) *Handler {
	return &Handler{servico: servico}
}

// Rotas registra as rotas públicas de autenticação.
func (h *Handler) Rotas(r chi.Router) {
	r.Post(ROTA_LOGIN, h.login)
	r.Post(ROTA_REFRESH, h.renovar)
	r.Post(ROTA_LOGOUT, h.logout)
}

// ----- Utilitários -----

func montarResposta(credenciais Credenciais) RespostaSessao {
	return RespostaSessao{
		Token:        credenciais.TokenAcesso,
		TokenRefresh: credenciais.TokenRefresh,
		ExpiraEm:     credenciais.ExpiraEm.Format(time.RFC3339),
		Tipo:         TIPO_BEARER,
	}
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

	credenciais, err := h.servico.Autenticar(entrada.Senha)
	if errors.Is(err, ErroNaoAutorizado) {
		resposta.Erro(w, http.StatusUnauthorized, "senha inválida")
		return
	}
	if err != nil {
		resposta.Erro(w, http.StatusInternalServerError, err.Error())
		return
	}

	resposta.JSON(w, http.StatusOK, montarResposta(credenciais))
}

func (h *Handler) renovar(w http.ResponseWriter, r *http.Request) {
	var entrada EntradaRefresh
	if err := resposta.LerJSON(r, &entrada); err != nil {
		resposta.Erro(w, http.StatusBadRequest, "corpo JSON inválido")
		return
	}
	if entrada.TokenRefresh == "" {
		resposta.Erro(w, http.StatusBadRequest, "refresh_token é obrigatório")
		return
	}

	credenciais, err := h.servico.Renovar(entrada.TokenRefresh)
	if errors.Is(err, ErroNaoAutorizado) {
		resposta.Erro(w, http.StatusUnauthorized, "refresh token inválido ou expirado")
		return
	}
	if err != nil {
		resposta.Erro(w, http.StatusInternalServerError, err.Error())
		return
	}

	resposta.JSON(w, http.StatusOK, montarResposta(credenciais))
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var entrada EntradaRefresh
	if err := resposta.LerJSON(r, &entrada); err != nil {
		resposta.Erro(w, http.StatusBadRequest, "corpo JSON inválido")
		return
	}

	h.servico.Revogar(entrada.TokenRefresh)
	resposta.JSON(w, http.StatusOK, map[string]bool{"encerrado": true})
}
