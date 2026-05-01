// internal/mspclouds/mspclouds_test.go
//
// Testes unitários que simulam a API MSP Clouds via httptest.Server.

package mspclouds

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ----- Testes do Servico -----

func TestServicoMspConfiguradoFalsoQuandoVazio(t *testing.T) {
	svc := NovoServico(nil, "", nil)

	if svc.Configurado() {
		t.Error("Configurado() deveria ser false sem chaves")
	}
}

func TestServicoMspConfiguradoVerdadeiroComChaves(t *testing.T) {
	svc := NovoServico([]string{"chave-1"}, "", nil)

	if !svc.Configurado() {
		t.Error("Configurado() deveria ser true com chave presente")
	}
}

func TestRefreshMspRejeitaServicoVazio(t *testing.T) {
	svc := NovoServico(nil, "", nil)

	if _, err := svc.AtualizarERetornar(); err == nil {
		t.Error("esperado erro quando não há chaves")
	}
}

func TestRefreshMspBemSucedidoPopulaCache(t *testing.T) {
	alertas := []Alerta{
		{
			"client":          "ACME Corp",
			"type":            "backup_failed",
			"product_keyword": "cloudbackup",
			"message": map[string]any{
				"login_name": "acme-srv01",
				"error":      "disk full",
			},
		},
	}
	servidor := servidorMspFake(t, alertas)
	defer servidor.Close()

	svc := NovoServico([]string{"chave-teste"}, servidor.URL, servidor.Client())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh falhou: %v", err)
	}
	if len(resultado.Data) != 1 {
		t.Fatalf("esperado 1 alerta, obtido %d", len(resultado.Data))
	}
	if resultado.Data[0]["client"] != "ACME Corp" {
		t.Errorf("client esperado 'ACME Corp', obtido %v", resultado.Data[0]["client"])
	}
	if len(resultado.Falhas) != 0 {
		t.Errorf("esperado zero falhas, obtido %v", resultado.Falhas)
	}
}

func TestRefreshMspFalhaTotalRetornaErro(t *testing.T) {
	servidor := servidorMspFake500(t)
	defer servidor.Close()

	svc := NovoServico([]string{"chave-ruim"}, servidor.URL, servidor.Client())

	_, err := svc.AtualizarERetornar()
	if err == nil {
		t.Fatal("esperado erro quando todas as instâncias falham")
	}
}

func TestRefreshMspParcialMantemUmaInstancia(t *testing.T) {
	servidorBom := servidorMspFake(t, []Alerta{
		{"client": "Boa", "type": "x"},
	})
	defer servidorBom.Close()

	svc := NovoServico([]string{"chave-ok", "chave-ruim"}, servidorBom.URL, servidorBom.Client())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh não deveria falhar: %v", err)
	}
	if len(resultado.Data) < 1 {
		t.Errorf("esperado ao menos 1 alerta, obtido %d", len(resultado.Data))
	}
}

// ----- Modo demo -----

func TestModoDemoMspConfiguraServicoMesmoSemChaves(t *testing.T) {
	svc := NovoServico(nil, "", nil)

	if svc.Configurado() {
		t.Fatal("pré-condição falhou: Configurado() deveria ser false antes do modo demo")
	}

	svc.AtivarModoDemo(MocksDemonstracao())

	if !svc.Configurado() {
		t.Error("Configurado() deveria ser true após AtivarModoDemo")
	}
}

func TestModoDemoMspRefreshDevolveMocksSemErro(t *testing.T) {
	svc := NovoServico(nil, "", nil)
	svc.AtivarModoDemo(MocksDemonstracao())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh em modo demo não deveria falhar: %v", err)
	}
	if len(resultado.Data) == 0 {
		t.Error("refresh em modo demo deveria devolver mocks no campo Data")
	}
}

func TestModoDemoMspCachePopuladoNoCarregar(t *testing.T) {
	svc := NovoServico(nil, "", nil)
	svc.AtivarModoDemo(MocksDemonstracao())

	alertas, _ := svc.CarregarCache()
	if len(alertas) == 0 {
		t.Error("CarregarCache deveria devolver mocks após AtivarModoDemo")
	}
}

// ----- Utilitários -----

func TestMascararChave(t *testing.T) {
	casos := map[string]string{
		"":                 "…",
		"ABC":              "…",
		"ABC123":           "…",
		"ABCDEFG":          "ABCDEF…",
		"2058-0A8B-6FE0":   "2058-0…",
		"abcdefghijklmnop": "abcdef…",
	}
	for entrada, esperado := range casos {
		if obtido := mascararChave(entrada); obtido != esperado {
			t.Errorf("mascararChave(%q): esperado %q, obtido %q", entrada, esperado, obtido)
		}
	}
}

// ----- Handler HTTP -----

func TestHandlerMspRefreshSemChavesRetorna400(t *testing.T) {
	svc := NovoServico(nil, "", nil)
	h := NovoHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mspclouds/refresh", nil)
	h.refresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtido %d", rec.Code)
	}
}

func TestHandlerMspListarRetornaCacheVazio(t *testing.T) {
	svc := NovoServico(nil, "", nil)
	h := NovoHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mspclouds", nil)
	h.listar(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtido %d", rec.Code)
	}

	var corpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta não é JSON: %v", err)
	}
	if _, ok := corpo["data"]; !ok {
		t.Error("resposta deveria ter campo 'data'")
	}
}

// ----- Mocks (handlers/listeners no final) -----

// servidorMspFake responde /api/v1/alerts com alertasRetornados.
func servidorMspFake(t *testing.T, alertasRetornados []Alerta) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, ROTA_ALERTAS) {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("api_key") == "" {
			http.Error(w, "api_key obrigatoria", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(alertasRetornados)
	}))
}

// servidorMspFake500 sempre retorna erro.
func servidorMspFake500(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
}
