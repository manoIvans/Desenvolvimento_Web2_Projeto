// internal/seguranca/seguranca_test.go
//
// Testes dos middlewares de segurança: cabeçalhos de resposta e rate
// limiting por IP (token bucket).

package seguranca

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ----- Cabeçalhos de segurança -----

func TestCabecalhosSegurancaDefineHeaders(t *testing.T) {
	handler := CabecalhosSeguranca(handlerOk())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	esperados := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Content-Security-Policy": CSP_PADRAO,
	}
	for chave, valor := range esperados {
		if obtido := rec.Header().Get(chave); obtido != valor {
			t.Errorf("header %q esperado %q, obtido %q", chave, valor, obtido)
		}
	}
}

// ----- Rate limiting -----

func TestLimitadorPermiteAteACapacidade(t *testing.T) {
	lim := NovoLimitador(3, 0) // sem recarga

	for tentativa := 1; tentativa <= 3; tentativa++ {
		if !lim.Permitir("ip") {
			t.Fatalf("tentativa %d deveria passar dentro da capacidade", tentativa)
		}
	}
	if lim.Permitir("ip") {
		t.Error("4ª tentativa deveria ser bloqueada (capacidade 3)")
	}
}

func TestLimitadorRecarregaComOTempo(t *testing.T) {
	agora := time.Now()
	lim := NovoLimitador(1, 1) // 1 token, recarrega 1/s
	lim.relogio = func() time.Time { return agora }

	if !lim.Permitir("ip") {
		t.Fatal("primeira tentativa deveria passar")
	}
	if lim.Permitir("ip") {
		t.Fatal("segunda tentativa imediata deveria ser bloqueada")
	}

	agora = agora.Add(time.Second)
	if !lim.Permitir("ip") {
		t.Error("após 1s o balde deveria recarregar e permitir")
	}
}

func TestLimitadorRemoveBaldesOciosos(t *testing.T) {
	agora := time.Now()
	lim := NovoLimitador(2, 1) // 2 tokens, recarrega 1/s → enche em 2s
	lim.relogio = func() time.Time { return agora }

	lim.Permitir("ocioso") // cria o balde do IP "ocioso"

	// Passa o intervalo de limpeza (e muito além do tempo de recarga): ao
	// processar um novo IP, a varredura deve descartar o balde ocioso.
	agora = agora.Add(INTERVALO_LIMPEZA + time.Minute)
	lim.Permitir("novo")

	lim.mu.Lock()
	defer lim.mu.Unlock()
	if _, existe := lim.baldes["ocioso"]; existe {
		t.Error("balde ocioso e recarregado deveria ter sido removido pela limpeza")
	}
	if _, existe := lim.baldes["novo"]; !existe {
		t.Error("balde ativo não deveria ser removido")
	}
}

func TestMiddlewareBloqueiaExcessoCom429(t *testing.T) {
	lim := NovoLimitador(1, 0)
	handler := lim.Middleware(handlerOk())

	primeira := requisitarDeIP(handler, "10.0.0.1:1111")
	if primeira.Code != http.StatusOK {
		t.Fatalf("1ª requisição esperada 200, obtida %d", primeira.Code)
	}

	segunda := requisitarDeIP(handler, "10.0.0.1:2222")
	if segunda.Code != http.StatusTooManyRequests {
		t.Errorf("2ª requisição do mesmo IP esperada 429, obtida %d", segunda.Code)
	}
	if segunda.Header().Get("Retry-After") == "" {
		t.Error("resposta 429 deveria incluir Retry-After")
	}
}

func TestMiddlewareIsolaPorIP(t *testing.T) {
	lim := NovoLimitador(1, 0)
	handler := lim.Middleware(handlerOk())

	requisitarDeIP(handler, "1.1.1.1:1")
	if bloqueado := requisitarDeIP(handler, "1.1.1.1:2"); bloqueado.Code != http.StatusTooManyRequests {
		t.Errorf("IP esgotado esperado 429, obtido %d", bloqueado.Code)
	}
	if outro := requisitarDeIP(handler, "2.2.2.2:1"); outro.Code != http.StatusOK {
		t.Errorf("IP distinto deveria ter balde próprio (200), obtido %d", outro.Code)
	}
}

// ----- Helpers (no final) -----

func requisitarDeIP(handler http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func handlerOk() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
