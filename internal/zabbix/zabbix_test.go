// internal/zabbix/zabbix_test.go
//
// Testes unitários que simulam a API Zabbix JSON-RPC via httptest.Server.

package zabbix

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"SignalHub/internal/config"
)

// ----- Testes do Servico -----

func TestServicoConfiguradoFalsoQuandoVazio(t *testing.T) {
	svc := NovoServico(nil, nil)

	if svc.Configurado() {
		t.Error("Configurado() deveria ser false sem instâncias")
	}
}

func TestServicoConfiguradoVerdadeiroComInstancias(t *testing.T) {
	svc := NovoServico([]config.InstanciaZabbix{
		{Nome: "x", URL: "http://x", APIKey: "k"},
	}, nil)

	if !svc.Configurado() {
		t.Error("Configurado() deveria ser true com instância presente")
	}
}

func TestRefreshRejeitaServicoVazio(t *testing.T) {
	svc := NovoServico(nil, nil)

	if _, err := svc.AtualizarERetornar(); err == nil {
		t.Error("esperado erro quando não há instâncias")
	}
}

func TestRefreshBemSucedidoPopulaCache(t *testing.T) {
	triggers := []triggerRaw{
		{
			TriggerID:   "1001",
			Description: "CPU acima de 90%",
			Priority:    "4",
			LastChange:  "1700000000",
			Hosts: []struct {
				HostID string `json:"hostid"`
				Host   string `json:"host"`
				Name   string `json:"name"`
			}{{HostID: "1", Host: "srv01", Name: "srv01"}},
			Groups: []struct {
				GroupID string `json:"groupid"`
				Name    string `json:"name"`
			}{{GroupID: "1", Name: "Produção"}},
		},
	}
	servidor := servidorZabbixFake(t, triggers, "6.4.0")
	defer servidor.Close()

	svc := NovoServico([]config.InstanciaZabbix{
		{Nome: "Teste", URL: servidor.URL, APIKey: "chave-teste"},
	}, servidor.Client())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh falhou: %v", err)
	}
	if len(resultado.Data) != 1 {
		t.Fatalf("esperado 1 problema, obtido %d", len(resultado.Data))
	}
	if resultado.Data[0].Evento != "CPU acima de 90%" {
		t.Errorf("evento esperado 'CPU acima de 90%%', obtido %q", resultado.Data[0].Evento)
	}
	if resultado.Data[0].Host != "srv01" {
		t.Errorf("host esperado 'srv01', obtido %q", resultado.Data[0].Host)
	}
	if resultado.Data[0].Instancia != "Teste" {
		t.Errorf("instância esperada 'Teste', obtida %q", resultado.Data[0].Instancia)
	}
	if resultado.Versoes["Teste"] != "6.4.0" {
		t.Errorf("versão esperada '6.4.0', obtida %q", resultado.Versoes["Teste"])
	}
	if len(resultado.Falhas) != 0 {
		t.Errorf("esperado zero falhas, obtido %v", resultado.Falhas)
	}
}

func TestRefreshFalhaTotalRetornaErro(t *testing.T) {
	servidor := servidorZabbixFake500(t)
	defer servidor.Close()

	svc := NovoServico([]config.InstanciaZabbix{
		{Nome: "Quebrada", URL: servidor.URL, APIKey: "k"},
	}, servidor.Client())

	_, err := svc.AtualizarERetornar()
	if err == nil {
		t.Fatal("esperado erro quando todas as instâncias falham")
	}
	if !strings.Contains(err.Error(), "falharam") {
		t.Errorf("mensagem deveria mencionar falha, obtido: %v", err)
	}
}

func TestRefreshParcialMantemUmaInstancia(t *testing.T) {
	servidorBom := servidorZabbixFake(t, []triggerRaw{
		{TriggerID: "2", Description: "Disco cheio", Priority: "3", LastChange: "1700000100",
			Hosts: []struct {
				HostID string `json:"hostid"`
				Host   string `json:"host"`
				Name   string `json:"name"`
			}{{HostID: "9", Host: "srv02", Name: "srv02"}}},
	}, "6.0.0")
	defer servidorBom.Close()

	servidorRuim := servidorZabbixFake500(t)
	defer servidorRuim.Close()

	svc := NovoServico([]config.InstanciaZabbix{
		{Nome: "ok", URL: servidorBom.URL, APIKey: "k1"},
		{Nome: "quebrado", URL: servidorRuim.URL, APIKey: "k2"},
	}, servidorBom.Client())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh parcial não deveria falhar: %v", err)
	}
	if len(resultado.Data) != 1 {
		t.Errorf("esperado 1 problema (da instância ok), obtido %d", len(resultado.Data))
	}
	if len(resultado.Falhas) != 1 {
		t.Errorf("esperado 1 falha (instância quebrada), obtido %d", len(resultado.Falhas))
	}
}

func TestCarregarCacheVazioAntesDoRefresh(t *testing.T) {
	svc := NovoServico([]config.InstanciaZabbix{
		{Nome: "x", URL: "http://x", APIKey: "k"},
	}, nil)

	problemas, falhas, versoes := svc.CarregarCache()
	if len(problemas) != 0 {
		t.Errorf("cache deveria estar vazio, obtido %d problemas", len(problemas))
	}
	if len(falhas) != 0 {
		t.Errorf("falhas deveriam estar vazias, obtido %d", len(falhas))
	}
	if len(versoes) != 0 {
		t.Errorf("versões deveriam estar vazias, obtido %d", len(versoes))
	}
}

// ----- Modo demo -----

func TestModoDemoConfiguraServicoMesmoSemInstancias(t *testing.T) {
	svc := NovoServico(nil, nil)

	if svc.Configurado() {
		t.Fatal("pré-condição falhou: Configurado() deveria ser false antes do modo demo")
	}

	svc.AtivarModoDemo(MocksDemonstracao())

	if !svc.Configurado() {
		t.Error("Configurado() deveria ser true após AtivarModoDemo")
	}
}

func TestModoDemoRefreshDevolveMocksSemErro(t *testing.T) {
	svc := NovoServico(nil, nil)
	svc.AtivarModoDemo(MocksDemonstracao())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh em modo demo não deveria falhar: %v", err)
	}
	if len(resultado.Data) == 0 {
		t.Error("refresh em modo demo deveria devolver mocks no campo Data")
	}
	if resultado.Versoes[NOME_INSTANCIA_DEMO] != VERSAO_DEMO {
		t.Errorf("versão demo esperada %q, obtida %q", VERSAO_DEMO, resultado.Versoes[NOME_INSTANCIA_DEMO])
	}
}

func TestModoDemoCachePopuladoNoCarregar(t *testing.T) {
	svc := NovoServico(nil, nil)
	svc.AtivarModoDemo(MocksDemonstracao())

	problemas, _, _ := svc.CarregarCache()
	if len(problemas) == 0 {
		t.Error("CarregarCache deveria devolver mocks após AtivarModoDemo")
	}
}

// ----- Conversores -----

func TestPrioridadeParaRotulo(t *testing.T) {
	casos := map[string]string{
		"0": "Not classified",
		"1": "Information",
		"2": "Warning",
		"3": "Average",
		"4": "High",
		"5": "Disaster",
		"9": "",
	}
	for entrada, esperado := range casos {
		if obtido := prioridadeParaRotulo(entrada); obtido != esperado {
			t.Errorf("prioridade %q: esperado %q, obtido %q", entrada, esperado, obtido)
		}
	}
}

func TestClockParaRFC3339(t *testing.T) {
	if saida := clockParaRFC3339("0"); saida != "" {
		t.Errorf("clock 0 deveria retornar vazio, obtido %q", saida)
	}
	if saida := clockParaRFC3339(""); saida != "" {
		t.Errorf("clock vazio deveria retornar vazio, obtido %q", saida)
	}
	if saida := clockParaRFC3339("abc"); saida != "" {
		t.Errorf("clock inválido deveria retornar vazio, obtido %q", saida)
	}
	// 1700000000 → 2023-11-14T22:13:20Z
	if saida := clockParaRFC3339("1700000000"); !strings.HasPrefix(saida, "2023-11-14T") {
		t.Errorf("clock 1700000000: esperado começar com '2023-11-14T', obtido %q", saida)
	}
}

// ----- Handler HTTP -----

func TestHandlerRefreshSemInstanciasRetorna400(t *testing.T) {
	svc := NovoServico(nil, nil)
	h := NovoHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/zabbix/refresh", nil)
	h.refresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtido %d", rec.Code)
	}
}

func TestHandlerListarRetornaCacheVazio(t *testing.T) {
	svc := NovoServico(nil, nil)
	h := NovoHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/zabbix", nil)
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

// servidorZabbixFake retorna um httptest.Server que responde trigger.get com
// triggersRetornados e apiinfo.version com versao.
func servidorZabbixFake(t *testing.T, triggersRetornados []triggerRaw, versao string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corpo, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("fake zabbix: falha ao ler body: %v", err)
		}

		var req apiReq
		if err := json.Unmarshal(corpo, &req); err != nil {
			t.Fatalf("fake zabbix: body inválido: %v", err)
		}

		var resultado any
		switch req.Method {
		case METODO_TRIGGER_GET:
			resultado = triggersRetornados
		case METODO_APIINFO_VERSION:
			resultado = versao
		default:
			http.Error(w, "metodo nao suportado", http.StatusNotFound)
			return
		}

		bytesResultado, _ := json.Marshal(resultado)
		resposta := apiResp{Result: bytesResultado}
		bytesResposta, _ := json.Marshal(resposta)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bytesResposta)
	}))
}

// servidorZabbixFake500 sempre retorna HTTP 500.
func servidorZabbixFake500(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
}
