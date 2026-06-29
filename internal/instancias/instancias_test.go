// internal/instancias/instancias_test.go
//
// Testes do CRUD de instâncias Zabbix/MSP, contas Acronis e do endpoint
// aninhado 1:N, usando o QuerierSimulado (sem PostgreSQL real).

package instancias

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"SignalHub/internal/banco/consultas"
	"SignalHub/internal/banco/consultas/simulado"
)

// ----- Helpers de teste -----

func montarRouter(servico *Servico) http.Handler {
	r := chi.NewRouter()
	NovoHandler(servico).Rotas(r)
	return r
}

func entradaZabbixValida() EntradaZabbix {
	return EntradaZabbix{
		Nome:   "Produção",
		URL:    "https://zabbix.exemplo.com",
		APIKey: "chave-secreta-123",
	}
}

// ----- Zabbix: golden path -----

func TestCriarZabbixComDadosValidos(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	instancia, err := servico.CriarZabbix(context.Background(), entradaZabbixValida())
	if err != nil {
		t.Fatalf("criar instância válida falhou: %v", err)
	}
	if instancia.ID == 0 {
		t.Error("instância criada deveria ter id atribuído")
	}
	if instancia.URL != "https://zabbix.exemplo.com" {
		t.Errorf("url incorreta: %q", instancia.URL)
	}
}

func TestAtualizarZabbixSobrescreveCampos(t *testing.T) {
	servico := NovoServico(simulado.Novo())
	contexto := context.Background()

	criada, _ := servico.CriarZabbix(contexto, entradaZabbixValida())

	atualizada, err := servico.AtualizarZabbix(contexto, criada.ID, EntradaZabbix{
		Nome:   "Homologação",
		URL:    "https://hml.exemplo.com",
		APIKey: "outra-chave",
	})
	if err != nil {
		t.Fatalf("atualizar falhou: %v", err)
	}
	if atualizada.Nome != "Homologação" {
		t.Errorf("nome não atualizado: %q", atualizada.Nome)
	}
}

// ----- Zabbix: bordas e falhas -----

func TestCriarZabbixURLInvalida(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	entrada := entradaZabbixValida()
	entrada.URL = "nao-e-uma-url"

	if _, err := servico.CriarZabbix(context.Background(), entrada); err == nil {
		t.Error("esperado erro de validação para url inválida")
	}
}

func TestCriarZabbixSemApiKey(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	entrada := entradaZabbixValida()
	entrada.APIKey = ""

	if _, err := servico.CriarZabbix(context.Background(), entrada); err == nil {
		t.Error("esperado erro de validação para api_key vazia")
	}
}

func TestRemoverZabbixInexistenteRetornaErro(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	if err := servico.RemoverZabbix(context.Background(), 999); err == nil {
		t.Error("esperado erro ao remover instância inexistente")
	}
}

// ----- Relacionamento 1:N (zabbix_instancias --< filtros) -----

func TestBuscarZabbixComFiltrosAninhados(t *testing.T) {
	q := simulado.Novo()
	servico := NovoServico(q)
	contexto := context.Background()

	instancia, _ := servico.CriarZabbix(contexto, entradaZabbixValida())

	// Dois filtros vinculados à instância criada.
	_, _ = q.CriarFiltro(contexto, consultas.CriarFiltroParams{
		InstanciaID: instancia.ID, Alvo: "hosts", Host: "srv01",
	})
	_, _ = q.CriarFiltro(contexto, consultas.CriarFiltroParams{
		InstanciaID: instancia.ID, Alvo: "eventos", Evento: "disco cheio",
	})

	resultado, err := servico.BuscarZabbixComFiltros(contexto, instancia.ID)
	if err != nil {
		t.Fatalf("busca aninhada falhou: %v", err)
	}
	if resultado.ID != instancia.ID {
		t.Errorf("instância errada: esperado id %d, obtido %d", instancia.ID, resultado.ID)
	}
	if len(resultado.Filtros) != 2 {
		t.Fatalf("esperado 2 filtros aninhados, obtido %d", len(resultado.Filtros))
	}
	if resultado.Filtros[0].InstanciaID != instancia.ID {
		t.Error("filtro aninhado deveria referenciar a instância dona")
	}
}

func TestRemoverZabbixRemoveFiltrosEmCascata(t *testing.T) {
	q := simulado.Novo()
	servico := NovoServico(q)
	contexto := context.Background()

	instancia, _ := servico.CriarZabbix(contexto, entradaZabbixValida())
	_, _ = q.CriarFiltro(contexto, consultas.CriarFiltroParams{
		InstanciaID: instancia.ID, Alvo: "hosts", Host: "srv01",
	})

	if err := servico.RemoverZabbix(contexto, instancia.ID); err != nil {
		t.Fatalf("remover instância falhou: %v", err)
	}

	restantes, _ := q.ListarFiltrosPorInstancia(contexto, instancia.ID)
	if len(restantes) != 0 {
		t.Errorf("filtros deveriam ter sido removidos em cascata, restaram %d", len(restantes))
	}
}

// ----- MSP Clouds -----

func TestCriarMspComDadosValidos(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	instancia, err := servico.CriarMsp(context.Background(), EntradaMsp{APIKey: "msp-chave-1"})
	if err != nil {
		t.Fatalf("criar instância MSP válida falhou: %v", err)
	}
	if instancia.APIKey != "msp-chave-1" {
		t.Errorf("api_key incorreta: %q", instancia.APIKey)
	}
}

func TestCriarMspSemApiKey(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	if _, err := servico.CriarMsp(context.Background(), EntradaMsp{APIKey: "  "}); err == nil {
		t.Error("esperado erro de validação para api_key vazia")
	}
}

// ----- Acronis -----

func entradaAcronisValida() EntradaAcronis {
	return EntradaAcronis{
		Nome:         "Tenant principal",
		ServerURL:    "https://eu2-cloud.acronis.com",
		ClientID:     "client-abc",
		ClientSecret: "secret-xyz",
	}
}

func TestCriarAcronisComDadosValidos(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	conta, err := servico.CriarAcronis(context.Background(), entradaAcronisValida())
	if err != nil {
		t.Fatalf("criar conta Acronis válida falhou: %v", err)
	}
	if conta.ID == 0 {
		t.Error("conta criada deveria ter id atribuído")
	}
	if conta.ClientID != "client-abc" {
		t.Errorf("client_id incorreto: %q", conta.ClientID)
	}
}

func TestCriarAcronisComLoginSemServerURL(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	entrada := EntradaAcronis{
		Login:        "operador@empresa",
		ClientID:     "client-abc",
		ClientSecret: "secret-xyz",
	}
	if _, err := servico.CriarAcronis(context.Background(), entrada); err != nil {
		t.Errorf("login deveria bastar como destino (descoberta), erro: %v", err)
	}
}

func TestCriarAcronisSemDestinoFalha(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	entrada := entradaAcronisValida()
	entrada.ServerURL = ""
	entrada.Login = ""

	if _, err := servico.CriarAcronis(context.Background(), entrada); err == nil {
		t.Error("esperado erro: sem server_url nem login não há destino")
	}
}

func TestCriarAcronisSemCredenciaisFalha(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	entrada := entradaAcronisValida()
	entrada.ClientSecret = ""

	if _, err := servico.CriarAcronis(context.Background(), entrada); err == nil {
		t.Error("esperado erro de validação para client_secret vazio")
	}
}

func TestCriarAcronisURLInvalidaFalha(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	entrada := entradaAcronisValida()
	entrada.ServerURL = "nao-e-uma-url"

	if _, err := servico.CriarAcronis(context.Background(), entrada); err == nil {
		t.Error("esperado erro de validação para server_url inválida")
	}
}

func TestAtualizarAcronisSobrescreveCampos(t *testing.T) {
	servico := NovoServico(simulado.Novo())
	contexto := context.Background()

	criada, _ := servico.CriarAcronis(contexto, entradaAcronisValida())

	atualizada, err := servico.AtualizarAcronis(contexto, criada.ID, EntradaAcronis{
		Nome:         "Tenant secundário",
		Login:        "outro@empresa",
		ClientID:     "client-def",
		ClientSecret: "secret-novo",
	})
	if err != nil {
		t.Fatalf("atualizar falhou: %v", err)
	}
	if atualizada.Nome != "Tenant secundário" {
		t.Errorf("nome não atualizado: %q", atualizada.Nome)
	}
	if atualizada.ServerURL != "" {
		t.Errorf("server_url deveria ter sido sobrescrito para vazio, obtido %q", atualizada.ServerURL)
	}
}

func TestRemoverAcronisInexistenteRetornaErro(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	if err := servico.RemoverAcronis(context.Background(), 999); err == nil {
		t.Error("esperado erro ao remover conta inexistente")
	}
}

// ----- Handler HTTP -----

func TestHandlerCriarZabbixRetorna201(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	corpo := `{"nome":"Produção","url":"https://zabbix.exemplo.com","api_key":"k1"}`
	req := httptest.NewRequest(http.MethodPost, "/zabbix/instancias", strings.NewReader(corpo))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("esperado 201, obtido %d (corpo: %s)", rec.Code, rec.Body.String())
	}
}

func TestHandlerCriarZabbixURLInvalidaRetorna400(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	corpo := `{"nome":"X","url":"ftp://errado","api_key":"k1"}`
	req := httptest.NewRequest(http.MethodPost, "/zabbix/instancias", strings.NewReader(corpo))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400 para url inválida, obtido %d", rec.Code)
	}
}

func TestHandlerBuscarZabbixInexistenteRetorna404(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	req := httptest.NewRequest(http.MethodGet, "/zabbix/instancias/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("esperado 404, obtido %d", rec.Code)
	}
}

func TestHandlerBuscarZabbixComFiltrosRetornaJSON(t *testing.T) {
	q := simulado.Novo()
	servico := NovoServico(q)
	router := montarRouter(servico)

	instancia, _ := servico.CriarZabbix(context.Background(), entradaZabbixValida())
	_, _ = q.CriarFiltro(context.Background(), consultas.CriarFiltroParams{
		InstanciaID: instancia.ID, Alvo: "hosts", Host: "srv01",
	})

	req := httptest.NewRequest(http.MethodGet, "/zabbix/instancias/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtido %d", rec.Code)
	}

	var corpo struct {
		ID      int32 `json:"id"`
		Filtros []any `json:"filtros"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta não é JSON: %v", err)
	}
	if len(corpo.Filtros) != 1 {
		t.Errorf("esperado 1 filtro aninhado no JSON, obtido %d", len(corpo.Filtros))
	}
}

func TestHandlerCriarMspRetorna201(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	req := httptest.NewRequest(http.MethodPost, "/mspclouds/instancias", strings.NewReader(`{"api_key":"msp-1"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("esperado 201, obtido %d (corpo: %s)", rec.Code, rec.Body.String())
	}
}

func TestHandlerCriarAcronisRetorna201(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	corpo := `{"nome":"Tenant","server_url":"https://eu2-cloud.acronis.com","client_id":"c1","client_secret":"s1"}`
	req := httptest.NewRequest(http.MethodPost, "/acronis/contas", strings.NewReader(corpo))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("esperado 201, obtido %d (corpo: %s)", rec.Code, rec.Body.String())
	}
}

func TestHandlerCriarAcronisSemDestinoRetorna400(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	corpo := `{"nome":"Tenant","client_id":"c1","client_secret":"s1"}`
	req := httptest.NewRequest(http.MethodPost, "/acronis/contas", strings.NewReader(corpo))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400 (sem server_url nem login), obtido %d", rec.Code)
	}
}
