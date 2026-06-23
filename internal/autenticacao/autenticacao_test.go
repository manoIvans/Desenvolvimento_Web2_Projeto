// internal/autenticacao/autenticacao_test.go
//
// Testes unitários do domínio Autenticação: serviço (login, rotação de
// refresh, validação de token), handlers POST /login, /refresh e /logout, e
// o middleware Proteger.

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
	SEGREDO_DEMO      = "segredo-de-teste-bem-grande-123"
)

// ----- Helpers de teste -----

func servicoDeTeste() *Servico {
	return NovoServico(Config{
		SegredoJWT:  []byte(SEGREDO_DEMO),
		TokenMestre: TOKEN_MESTRE_DEMO,
		SenhaLogin:  SENHA_DEMO,
		TTLAcesso:   time.Hour,
		TTLRefresh:  time.Hour,
	})
}

// ----- Testes do Servico -----

func TestAutenticarSenhaCorretaEmiteCredenciais(t *testing.T) {
	svc := servicoDeTeste()

	credenciais, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("Autenticar não deveria falhar: %v", err)
	}
	if credenciais.TokenAcesso == "" {
		t.Error("access token emitido deveria ser não-vazio")
	}
	if credenciais.TokenRefresh == "" {
		t.Error("refresh token emitido deveria ser não-vazio")
	}
	if !credenciais.ExpiraEm.After(time.Now()) {
		t.Error("expiração deveria ser futura")
	}
}

func TestAutenticarSenhaErradaRetornaNaoAutorizado(t *testing.T) {
	svc := servicoDeTeste()

	_, err := svc.Autenticar("senha-errada")
	if !errors.Is(err, ErroNaoAutorizado) {
		t.Errorf("esperado ErroNaoAutorizado, obtido %v", err)
	}
}

func TestTokenValidoAceitaMestre(t *testing.T) {
	svc := servicoDeTeste()

	if !svc.TokenValido(TOKEN_MESTRE_DEMO) {
		t.Error("token mestre deveria ser válido")
	}
}

func TestTokenValidoAceitaAccessTokenFresco(t *testing.T) {
	svc := servicoDeTeste()

	credenciais, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}
	if !svc.TokenValido(credenciais.TokenAcesso) {
		t.Error("access token recém-emitido deveria ser válido")
	}
}

func TestTokenValidoRejeitaTokenAleatorio(t *testing.T) {
	svc := servicoDeTeste()

	if svc.TokenValido("token-aleatorio-nunca-emitido") {
		t.Error("token desconhecido não deveria ser válido")
	}
}

func TestTokenValidoRejeitaRefreshTokenComoAcesso(t *testing.T) {
	svc := servicoDeTeste()

	credenciais, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}
	// O refresh token é opaco (não é JWT) — não pode passar como Bearer.
	if svc.TokenValido(credenciais.TokenRefresh) {
		t.Error("refresh token não deveria ser aceito como access token")
	}
}

func TestRenovarRotacionaRefreshToken(t *testing.T) {
	svc := servicoDeTeste()

	primeira, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}

	segunda, err := svc.Renovar(primeira.TokenRefresh)
	if err != nil {
		t.Fatalf("Renovar não deveria falhar: %v", err)
	}
	if segunda.TokenRefresh == primeira.TokenRefresh {
		t.Error("refresh token deveria ser rotacionado (novo valor)")
	}
	if !svc.TokenValido(segunda.TokenAcesso) {
		t.Error("novo access token deveria ser válido")
	}

	// O refresh antigo deve ter sido invalidado pela rotação.
	if _, err := svc.Renovar(primeira.TokenRefresh); !errors.Is(err, ErroNaoAutorizado) {
		t.Errorf("refresh rotacionado deveria ser inválido, obtido %v", err)
	}
}

func TestRenovarTokenInvalidoRetornaNaoAutorizado(t *testing.T) {
	svc := servicoDeTeste()

	if _, err := svc.Renovar("refresh-inexistente"); !errors.Is(err, ErroNaoAutorizado) {
		t.Errorf("esperado ErroNaoAutorizado, obtido %v", err)
	}
}

func TestRenovarRefreshExpiradoRetornaNaoAutorizado(t *testing.T) {
	svc := NovoServico(Config{
		SegredoJWT:  []byte(SEGREDO_DEMO),
		TokenMestre: TOKEN_MESTRE_DEMO,
		SenhaLogin:  SENHA_DEMO,
		TTLAcesso:   time.Hour,
		TTLRefresh:  time.Millisecond,
	})

	credenciais, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	if _, err := svc.Renovar(credenciais.TokenRefresh); !errors.Is(err, ErroNaoAutorizado) {
		t.Errorf("refresh expirado deveria ser inválido, obtido %v", err)
	}
}

func TestRevogarInvalidaRefreshToken(t *testing.T) {
	svc := servicoDeTeste()

	credenciais, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}
	svc.Revogar(credenciais.TokenRefresh)

	if _, err := svc.Renovar(credenciais.TokenRefresh); !errors.Is(err, ErroNaoAutorizado) {
		t.Errorf("refresh revogado deveria ser inválido, obtido %v", err)
	}
}

// ----- Handlers /login, /refresh, /logout -----

func TestHandlerLoginRetornaCredenciais(t *testing.T) {
	h := NovoHandler(servicoDeTeste())

	resposta := executarJSON(t, h.login, http.MethodPost, ROTA_LOGIN, EntradaLogin{Senha: SENHA_DEMO})
	if resposta.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtido %d", resposta.Code)
	}

	var sessao RespostaSessao
	decodificar(t, resposta, &sessao)
	if sessao.Token == "" || sessao.TokenRefresh == "" {
		t.Error("login deveria devolver token e refresh_token")
	}
	if sessao.Tipo != TIPO_BEARER {
		t.Errorf("tipo esperado %q, obtido %q", TIPO_BEARER, sessao.Tipo)
	}
}

func TestHandlerLoginSenhaErradaRetorna401(t *testing.T) {
	h := NovoHandler(servicoDeTeste())

	resposta := executarJSON(t, h.login, http.MethodPost, ROTA_LOGIN, EntradaLogin{Senha: "errada"})
	if resposta.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtido %d", resposta.Code)
	}
}

func TestHandlerLoginSenhaVaziaRetorna400(t *testing.T) {
	h := NovoHandler(servicoDeTeste())

	resposta := executarJSON(t, h.login, http.MethodPost, ROTA_LOGIN, EntradaLogin{Senha: ""})
	if resposta.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtido %d", resposta.Code)
	}
}

func TestHandlerRenovarRetornaNovasCredenciais(t *testing.T) {
	svc := servicoDeTeste()
	h := NovoHandler(svc)

	credenciais, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}

	resposta := executarJSON(t, h.renovar, http.MethodPost, ROTA_REFRESH, EntradaRefresh{TokenRefresh: credenciais.TokenRefresh})
	if resposta.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtido %d", resposta.Code)
	}

	var sessao RespostaSessao
	decodificar(t, resposta, &sessao)
	if !svc.TokenValido(sessao.Token) {
		t.Error("token renovado deveria ser válido")
	}
}

func TestHandlerRenovarTokenInvalidoRetorna401(t *testing.T) {
	h := NovoHandler(servicoDeTeste())

	resposta := executarJSON(t, h.renovar, http.MethodPost, ROTA_REFRESH, EntradaRefresh{TokenRefresh: "invalido"})
	if resposta.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtido %d", resposta.Code)
	}
}

func TestHandlerLogoutRevogaRefresh(t *testing.T) {
	svc := servicoDeTeste()
	h := NovoHandler(svc)

	credenciais, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}

	resposta := executarJSON(t, h.logout, http.MethodPost, ROTA_LOGOUT, EntradaRefresh{TokenRefresh: credenciais.TokenRefresh})
	if resposta.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtido %d", resposta.Code)
	}
	if _, err := svc.Renovar(credenciais.TokenRefresh); !errors.Is(err, ErroNaoAutorizado) {
		t.Error("refresh deveria estar revogado após logout")
	}
}

// ----- Middleware Proteger -----

func TestMiddlewareSemHeaderRetorna401(t *testing.T) {
	protegido := Proteger(servicoDeTeste())(handlerOkDeTeste())

	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	rec := httptest.NewRecorder()
	protegido.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401 sem header, obtido %d", rec.Code)
	}
}

func TestMiddlewareTokenMestreLiberaAcesso(t *testing.T) {
	protegido := Proteger(servicoDeTeste())(handlerOkDeTeste())

	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	req.Header.Set(HEADER_AUTORIZACAO, PREFIXO_BEARER+TOKEN_MESTRE_DEMO)
	rec := httptest.NewRecorder()
	protegido.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200 com token mestre, obtido %d (body: %s)", rec.Code, rec.Body)
	}
}

func TestMiddlewareAccessTokenLiberaAcesso(t *testing.T) {
	svc := servicoDeTeste()
	protegido := Proteger(svc)(handlerOkDeTeste())

	credenciais, err := svc.Autenticar(SENHA_DEMO)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	req.Header.Set(HEADER_AUTORIZACAO, PREFIXO_BEARER+credenciais.TokenAcesso)
	rec := httptest.NewRecorder()
	protegido.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200 com access token, obtido %d (body: %s)", rec.Code, rec.Body)
	}
}

func TestMiddlewareTokenInvalidoRetorna401(t *testing.T) {
	protegido := Proteger(servicoDeTeste())(handlerOkDeTeste())

	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	req.Header.Set(HEADER_AUTORIZACAO, PREFIXO_BEARER+"chave-invalida")
	rec := httptest.NewRecorder()
	protegido.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401 com token inválido, obtido %d", rec.Code)
	}
}

func TestMiddlewareSemPrefixoBearerRetorna401(t *testing.T) {
	protegido := Proteger(servicoDeTeste())(handlerOkDeTeste())

	req := httptest.NewRequest(http.MethodGet, "/protegido", nil)
	req.Header.Set(HEADER_AUTORIZACAO, TOKEN_MESTRE_DEMO) // faltando "Bearer "
	rec := httptest.NewRecorder()
	protegido.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401 sem prefixo Bearer, obtido %d", rec.Code)
	}
}

// ----- Mocks e helpers (no final) -----

func executarJSON(t *testing.T, handler http.HandlerFunc, metodo, rota string, corpo any) *httptest.ResponseRecorder {
	t.Helper()
	bruto, err := json.Marshal(corpo)
	if err != nil {
		t.Fatalf("falha ao serializar corpo: %v", err)
	}
	req := httptest.NewRequest(metodo, rota, bytes.NewReader(bruto))
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func decodificar(t *testing.T, rec *httptest.ResponseRecorder, destino any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), destino); err != nil {
		t.Fatalf("resposta inválida: %v", err)
	}
}

func handlerOkDeTeste() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = strings.NewReader("ok").WriteTo(w)
	})
}
