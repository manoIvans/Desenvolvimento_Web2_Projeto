// internal/resposta/resposta.go
//
// Helpers HTTP compartilhados pelos domínios persistidos (instancias,
// filtros): serialização JSON, respostas de erro e leitura de parâmetros.

package resposta

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ----- Constantes -----

const CONTENT_TYPE_JSON = "application/json; charset=utf-8"

// ----- Erros -----

// ErroIDInvalido indica um parâmetro {id} de rota ausente ou não numérico.
var ErroIDInvalido = errors.New("id inválido")

// ----- API pública -----

// JSON escreve v como JSON com o status informado.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", CONTENT_TYPE_JSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Erro escreve uma resposta de erro padronizada {"error": msg}.
func Erro(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

// LerJSON decodifica o corpo da requisição em destino.
func LerJSON(r *http.Request, destino any) error {
	defer r.Body.Close()
	decodificador := json.NewDecoder(r.Body)
	decodificador.DisallowUnknownFields()
	return decodificador.Decode(destino)
}

// IDdaRota extrai o parâmetro {id} da URL e o converte para int32.
// Retorna ErroIDInvalido quando ausente, não numérico ou não-positivo.
func IDdaRota(r *http.Request, nomeParam string) (int32, error) {
	bruto := chi.URLParam(r, nomeParam)
	if bruto == "" {
		return 0, ErroIDInvalido
	}

	valor, err := strconv.ParseInt(bruto, 10, 32)
	if err != nil || valor <= 0 {
		return 0, ErroIDInvalido
	}
	return int32(valor), nil
}
