// internal/resposta/erros.go
//
// Classificação de erros de serviço em respostas HTTP. Centraliza o
// mapeamento erro → status para todos os domínios persistidos.

package resposta

import (
	"errors"
	"net/http"

	"SignalHub/internal/banco"
)

// ----- Tipo ErroValidacao -----

// ErroValidacao representa entrada inválida vinda do cliente (HTTP 400).
type ErroValidacao struct {
	Mensagem string
}

func (e ErroValidacao) Error() string {
	return e.Mensagem
}

// Validacao constrói um ErroValidacao com a mensagem informada.
func Validacao(mensagem string) error {
	return ErroValidacao{Mensagem: mensagem}
}

// ----- Tratamento -----

// Tratar mapeia um erro de serviço para a resposta HTTP adequada:
// validação → 400, não encontrado → 404, conflito UNIQUE → 409,
// violação de FK → 400, qualquer outro → 500.
func Tratar(w http.ResponseWriter, err error) {
	var validacao ErroValidacao
	switch {
	case errors.As(err, &validacao):
		Erro(w, http.StatusBadRequest, err.Error())
	case banco.EhNaoEncontrado(err):
		Erro(w, http.StatusNotFound, "registro não encontrado")
	case banco.EhConflito(err):
		Erro(w, http.StatusConflict, "registro já existe (valor único duplicado)")
	case banco.EhViolacaoFK(err):
		Erro(w, http.StatusBadRequest, "referência a registro inexistente")
	default:
		Erro(w, http.StatusInternalServerError, err.Error())
	}
}
