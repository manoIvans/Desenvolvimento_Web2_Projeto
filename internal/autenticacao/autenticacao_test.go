// internal/autenticacao/autenticacao_test.go
//
// Testes unitários do domínio Autenticação: serviço de tokens, handler
// POST /login e middleware Proteger.

package autenticacao

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ----- Constantes de teste -----

const (
	SENHA_DEMO        = "umasenhacriativa"
	TOKEN_MESTRE_DEMO = "mestre-imortal"
)

// ----- Testes do Servico -----

func TestAutenticarSenhaCorretaEmiteToken(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)

	token, expiraEm, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("Autenticar não deveria falhar: %v", err)
	}
	if token == "" {
		t.Error("token emitido deveria ser não-vazio")
	}
	if !expiraEm.After(time.Now()) {
		t.Error("expiração deveria ser futura")
	}
}

func TestAutenticarSenhaErradaRetornaNaoAutorizado(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)

	_, _, err := svc.Autenticar("senha-errada")
	if !errors.Is(err, ErroNaoAutorizado) {
		t.Errorf("esperado ErroNaoAutorizado, obtido %v", err)
	}
}

func TestTokenValidoAceitaMestre(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)

	if !svc.TokenValido(TOKEN_MESTRE_DEMO) {
		t.Error("token mestre deveria ser válido")
	}
}

func TestTokenValidoAceitaSessaoFresca(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)

	token, _, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}
	if !svc.TokenValido(token) {
		t.Error("token recém-emitido deveria ser válido")
	}
}

func TestTokenValidoRejeitaTokenAleatorio(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)

	if svc.TokenValido("token-aleatorio-nunca-emitido") {
		t.Error("token desconhecido não deveria ser válido")
	}
}

func TestTokenValidoRejeitaSessaoExpirada(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Millisecond)

	token, _, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	if svc.TokenValido(token) {
		t.Error("token deveria ter expirado")
	}
}

func TestRevogarRemoveSessao(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)

	token, _, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}
	svc.Revogar(token)

	if svc.TokenValido(token) {
		t.Error("sessão revogada não deveria ser válida")
	}
}

func TestRevogarNaoAfetaTokenMestre(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)
	svc.Revogar(TOKEN_MESTRE_DEMO)

	if !svc.TokenValido(TOKEN_MESTRE_DEMO) {
		t.Error("token mestre não deveria ser revogável")
	}
}

// ----- Handler POST /login -----

func TestHandlerLoginRetornaToken(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)
	h := NovoHandler(svc)

	corpo, _ := json.Marshal(EntradaLogin{Senha: SENHA_DEMO})
	req := httptest.NewRequest(http.MethodPost, ROTA_LOGIN, bytes.NewReader(corpo))
	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtido %d", rec.Code)
	}
	var resposta RespostaLogin
	if err := json.Unmarshal(rec.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("resposta inválida: %v", err)
	}
	if resposta.Token == "" {
		t.Error("token deveria estar presente na resposta")
	}
	if !svc.TokenValido(resposta.Token) {
		t.Error("token devolvido deveria ser aceito por TokenValido")
	}
}

func TestHandlerLoginSenhaErradaRetorna401(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)
	h := NovoHandler(svc)

	corpo, _ := json.Marshal(EntradaLogin{Senha: "errada"})
	req := httptest.NewRequest(http.MethodPost, ROTA_LOGIN, bytes.NewReader(corpo))
	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtido %d", rec.Code)
	}
}

func TestHandlerLoginSenhaVaziaRetorna400(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)
	h := NovoHandler(svc)

	corpo, _ := json.Marshal(EntradaLogin{Senha: ""})
	req := httptest.NewRequest(http.MethodPost, ROTA_LOGIN, bytes.NewReader(corpo))
	rec := httptest.NewRecorder()
	h.login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtido %d", rec.Code)
	}
}

// ----- Middleware Proteger -----

func TestMiddlewareSemHeaderRetorna401(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)
	protegido := Proteger(svc)(handlerOkDeTeste())

	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	rec := httptest.NewRecorder()
	protegido.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401 sem header, obtido %d", rec.Code)
	}
}

func TestMiddlewareTokenMestreLiberaAcesso(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)
	protegido := Proteger(svc)(handlerOkDeTeste())

	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	req.Header.Set(HEADER_AUTORIZACAO, PREFIXO_BEARER+TOKEN_MESTRE_DEMO)
	rec := httptest.NewRecorder()
	protegido.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200 com token mestre, obtido %d (body: %s)", rec.Code, rec.Body)
	}
}

func TestMiddlewareTokenInvalidoRetorna401(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)
	protegido := Proteger(svc)(handlerOkDeTeste())

	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	req.Header.Set(HEADER_AUTORIZACAO, PREFIXO_BEARER+"chave-invalida")
	rec := httptest.NewRecorder()
	protegido.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401 com token inválido, obtido %d", rec.Code)
	}
}

func TestMiddlewareSemPrefixoBearerRetorna401(t *testing.T) {
	svc := NovoServico(TOKEN_MESTRE_DEMO, SENHA_DEMO, time.Hour)
	protegido := Proteger(svc)(handlerOkDeTeste())

	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	req.Header.Set(HEADER_AUTORIZACAO, TOKEN_MESTRE_DEMO) // faltando "Bearer "
	rec := httptest.NewRecorder()
	protegido.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401 sem prefixo Bearer, obtido %d", rec.Code)
	}
}

// ----- Mocks (handlers no final) -----

func handlerOkDeTeste() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = strings.NewReader("ok").WriteTo(w)
	})
}
