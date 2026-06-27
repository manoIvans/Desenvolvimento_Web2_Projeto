// internal/servidor/servidor_test.go
//
// Testes de integração do roteador completo (httptest, sem TCP nem banco):
// rotas públicas, gate de autenticação, fluxo login → acesso → refresh com
// rotação, e presença dos cabeçalhos de segurança.

package servidor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"SignalHub/internal/autenticacao"
	"SignalHub/internal/banco/consultas/simulado"
	"SignalHub/internal/filtros"
	"SignalHub/internal/instancias"
	"SignalHub/internal/mspclouds"
	"SignalHub/internal/zabbix"
)

// ----- Constantes de teste -----

const (
	SENHA_DEMO   = "senha-de-teste"
	SEGREDO_DEMO = "segredo-jwt-de-teste-grande"
)

// ----- Setup -----

func montarRouterDeTeste() (http.Handler, *autenticacao.Servico) {
	bd := simulado.Novo()
	servicoAuth := autenticacao.NovoServico(autenticacao.Config{
		SegredoJWT: []byte(SEGREDO_DEMO),
		SenhaLogin: SENHA_DEMO,
		TTLAcesso:  time.Hour,
		TTLRefresh: time.Hour,
	})

	router := MontarRouter(Dependencias{
		HandlerZabbix:       zabbix.NovoHandler(zabbix.NovoServico(nil, nil)),
		HandlerMsp:          mspclouds.NovoHandler(mspclouds.NovoServico(nil, "", nil)),
		HandlerInstancias:   instancias.NovoHandler(instancias.NovoServico(bd)),
		HandlerFiltros:      filtros.NovoHandler(filtros.NovoServico(bd)),
		HandlerAutenticacao: autenticacao.NovoHandler(servicoAuth),
		ServicoAutenticacao: servicoAuth,
	})
	return router, servicoAuth
}

// ----- Testes -----

func TestHealthzEhPublicoComCabecalhosDeSeguranca(t *testing.T) {
	router, _ := montarRouterDeTeste()

	resposta := requisitar(router, http.MethodGet, "/healthz", "", nil)
	if resposta.Code != http.StatusOK {
		t.Fatalf("/healthz esperado 200, obtido %d", resposta.Code)
	}
	if resposta.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("resposta deveria conter cabeçalhos de segurança")
	}
}

func TestRotaProtegidaSemTokenRetorna401(t *testing.T) {
	router, _ := montarRouterDeTeste()

	resposta := requisitar(router, http.MethodGet, "/zabbix/instancias", "", nil)
	if resposta.Code != http.StatusUnauthorized {
		t.Errorf("rota protegida sem token esperada 401, obtida %d", resposta.Code)
	}
}

func TestLoginSenhaErradaRetorna401(t *testing.T) {
	router, _ := montarRouterDeTeste()

	resposta := requisitar(router, http.MethodPost, "/login", "", map[string]string{"senha": "errada"})
	if resposta.Code != http.StatusUnauthorized {
		t.Errorf("login com senha errada esperado 401, obtido %d", resposta.Code)
	}
}

func TestFluxoLoginAcessoERefresh(t *testing.T) {
	router, _ := montarRouterDeTeste()

	// 1. Login válido devolve o par de tokens.
	login := requisitar(router, http.MethodPost, "/login", "", map[string]string{"senha": SENHA_DEMO})
	if login.Code != http.StatusOK {
		t.Fatalf("login esperado 200, obtido %d", login.Code)
	}
	sessao := decodificarSessao(t, login)
	if sessao.Token == "" || sessao.TokenRefresh == "" {
		t.Fatal("login deveria devolver token e refresh_token")
	}

	// 2. O access token libera a rota protegida.
	acesso := requisitar(router, http.MethodGet, "/zabbix/instancias", sessao.Token, nil)
	if acesso.Code != http.StatusOK {
		t.Fatalf("acesso com token esperado 200, obtido %d (body: %s)", acesso.Code, acesso.Body)
	}

	// 3. O refresh emite um novo par e invalida o anterior (rotação).
	renov := requisitar(router, http.MethodPost, "/refresh", "", map[string]string{"refresh_token": sessao.TokenRefresh})
	if renov.Code != http.StatusOK {
		t.Fatalf("refresh esperado 200, obtido %d", renov.Code)
	}
	if reuso := requisitar(router, http.MethodPost, "/refresh", "", map[string]string{"refresh_token": sessao.TokenRefresh}); reuso.Code != http.StatusUnauthorized {
		t.Errorf("reuso do refresh rotacionado esperado 401, obtido %d", reuso.Code)
	}
}

// ----- Helpers (no final) -----

func requisitar(router http.Handler, metodo, rota, token string, corpo any) *httptest.ResponseRecorder {
	var leitor *bytes.Reader
	if corpo != nil {
		bruto, _ := json.Marshal(corpo)
		leitor = bytes.NewReader(bruto)
	} else {
		leitor = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(metodo, rota, leitor)
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodificarSessao(t *testing.T, rec *httptest.ResponseRecorder) autenticacao.RespostaSessao {
	t.Helper()
	var sessao autenticacao.RespostaSessao
	if err := json.Unmarshal(rec.Body.Bytes(), &sessao); err != nil {
		t.Fatalf("resposta de sessão inválida: %v", err)
	}
	return sessao
}
