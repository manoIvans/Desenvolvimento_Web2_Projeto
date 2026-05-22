// internal/autenticacao/middleware.go
//
// Middleware HTTP que exige Bearer token válido. Sem token, com token
// desconhecido ou expirado a resposta é 401 — formato {"error": "..."}.

package autenticacao

import (
	"net/http"
	"strings"

	"SignalHub/internal/resposta"
)

// ----- Constantes -----

const (
	HEADER_AUTORIZACAO = "Authorization"
	PREFIXO_BEARER     = "Bearer "
)

// ----- API pública -----

// Proteger devolve um middleware Chi que valida o header Authorization.
// Aplique apenas ao subgrupo de rotas que exige autenticação.
func Proteger(servico *Servico) func(http.Handler) http.Handler {
	return func(proximo http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extrairToken(r)
			if !servico.TokenValido(token) {
				resposta.Erro(w, http.StatusUnauthorized, "token ausente, inválido ou expirado")
				return
			}
			proximo.ServeHTTP(w, r)
		})
	}
}

// ----- Internos -----

func extrairToken(r *http.Request) string {
	cabecalho := r.Header.Get(HEADER_AUTORIZACAO)
	if !strings.HasPrefix(cabecalho, PREFIXO_BEARER) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(cabecalho, PREFIXO_BEARER))
}
