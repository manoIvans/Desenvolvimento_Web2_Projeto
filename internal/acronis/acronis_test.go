// internal/acronis/acronis_test.go
//
// Testes do domínio Acronis: estado de configuração, modo demo, conversão de
// alertas e o fluxo real (descoberta → token OAuth2 → alertas) simulado via
// httptest.Server, incluindo cache/reemissão de token e recuperação de 401.

package acronis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ----- Constantes de teste -----

const (
	CLIENT_ID_DEMO     = "cliente-demo"
	CLIENT_SECRET_DEMO = "segredo-demo"
)

// ----- Configurado -----

func TestConfiguradoFalsoQuandoVazio(t *testing.T) {
	if NovoServico(nil, nil).Configurado() {
		t.Error("Configurado() deveria ser false sem credenciais")
	}
}

func TestConfiguradoFalsoSemCredenciais(t *testing.T) {
	svc := NovoServico([]Conta{{ServerURL: "https://eu2-cloud.acronis.com"}}, nil)
	if svc.Configurado() {
		t.Error("Configurado() deveria ser false sem client_id/secret")
	}
}

func TestConfiguradoVerdadeiroComURLeCredenciais(t *testing.T) {
	svc := NovoServico([]Conta{{
		ServerURL:    "https://eu2-cloud.acronis.com",
		ClientID:     CLIENT_ID_DEMO,
		ClientSecret: CLIENT_SECRET_DEMO,
	}}, nil)
	if !svc.Configurado() {
		t.Error("Configurado() deveria ser true com URL + credenciais")
	}
}

func TestConfiguradoVerdadeiroComLoginParaDescoberta(t *testing.T) {
	svc := NovoServico([]Conta{{
		Login:        "operador@empresa",
		ClientID:     CLIENT_ID_DEMO,
		ClientSecret: CLIENT_SECRET_DEMO,
	}}, nil)
	if !svc.Configurado() {
		t.Error("Configurado() deveria ser true com login (descoberta) + credenciais")
	}
}

func TestRefreshRejeitaServicoVazio(t *testing.T) {
	if _, err := NovoServico(nil, nil).AtualizarERetornar(); err == nil {
		t.Error("esperado erro quando não há conta configurada")
	}
}

// ----- Modo demo -----

func TestModoDemoConfiguraServico(t *testing.T) {
	svc := NovoServico(nil, nil)
	if svc.Configurado() {
		t.Fatal("pré-condição: deveria estar não-configurado")
	}

	svc.AtivarModoDemo(MocksDemonstracao())
	if !svc.Configurado() {
		t.Error("Configurado() deveria ser true após AtivarModoDemo")
	}
}

func TestModoDemoRefreshDevolveMocks(t *testing.T) {
	svc := NovoServico(nil, nil)
	svc.AtivarModoDemo(MocksDemonstracao())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh em modo demo não deveria falhar: %v", err)
	}
	if len(resultado.Data) == 0 {
		t.Error("refresh em modo demo deveria devolver mocks")
	}
}

// ----- Conversão -----

func TestConverterAlerta(t *testing.T) {
	bruto := exemploAlertaRaw()

	alerta := converterAlerta(bruto)
	if alerta.Titulo != "Malware Detected" {
		t.Errorf("titulo esperado 'Malware Detected', obtido %q", alerta.Titulo)
	}
	if alerta.Categoria != "Protection" {
		t.Errorf("categoria deveria preferir details.category ('Protection'), obtido %q", alerta.Categoria)
	}
	if alerta.Severidade != "warning" {
		t.Errorf("severidade esperada 'warning', obtido %q", alerta.Severidade)
	}
	if alerta.Tenant != "286" {
		t.Errorf("tenant esperado '286', obtido %q", alerta.Tenant)
	}
	if alerta.Horario != "2023-11-07T14:39:57.556577425Z" {
		t.Errorf("horario deveria vir de createdAt, obtido %q", alerta.Horario)
	}
	if alerta.Source != SOURCE_API_ACRONIS {
		t.Errorf("source esperado %q, obtido %q", SOURCE_API_ACRONIS, alerta.Source)
	}
}

// ----- Fluxo real (httptest) -----

func TestFluxoLoginEAlertas(t *testing.T) {
	fake := novoFakeAcronis([]alertaRaw{exemploAlertaRaw()})
	defer fake.servidor.Close()

	svc := NovoServico([]Conta{{
		ServerURL:    fake.servidor.URL,
		ClientID:     CLIENT_ID_DEMO,
		ClientSecret: CLIENT_SECRET_DEMO,
	}}, fake.servidor.Client())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh falhou: %v", err)
	}
	if len(resultado.Data) != 1 {
		t.Fatalf("esperado 1 alerta, obtido %d", len(resultado.Data))
	}
	if resultado.Data[0].Titulo != "Malware Detected" {
		t.Errorf("alerta convertido incorreto: %+v", resultado.Data[0])
	}
	if n := fake.tokens.Load(); n != 1 {
		t.Errorf("esperado 1 emissão de token, obtido %d", n)
	}
	if !fake.limitEnviado.Load() {
		t.Error("a requisição de alertas deveria enviar o parâmetro limit")
	}
}

func TestTokenReutilizadoEntreRefreshes(t *testing.T) {
	fake := novoFakeAcronis([]alertaRaw{exemploAlertaRaw()})
	defer fake.servidor.Close()

	svc := NovoServico([]Conta{{
		ServerURL:    fake.servidor.URL,
		ClientID:     CLIENT_ID_DEMO,
		ClientSecret: CLIENT_SECRET_DEMO,
	}}, fake.servidor.Client())

	if _, err := svc.AtualizarERetornar(); err != nil {
		t.Fatalf("primeiro refresh falhou: %v", err)
	}
	if _, err := svc.AtualizarERetornar(); err != nil {
		t.Fatalf("segundo refresh falhou: %v", err)
	}
	if n := fake.tokens.Load(); n != 1 {
		t.Errorf("token deveria ser reutilizado (1 emissão), obtido %d", n)
	}
}

func TestTokenReemitidoApos401(t *testing.T) {
	fake := novoFakeAcronis([]alertaRaw{exemploAlertaRaw()})
	fake.alertas401UmaVez.Store(true)
	defer fake.servidor.Close()

	svc := NovoServico([]Conta{{
		ServerURL:    fake.servidor.URL,
		ClientID:     CLIENT_ID_DEMO,
		ClientSecret: CLIENT_SECRET_DEMO,
	}}, fake.servidor.Client())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh deveria recuperar do 401: %v", err)
	}
	if len(resultado.Data) != 1 {
		t.Fatalf("esperado 1 alerta após reemissão, obtido %d", len(resultado.Data))
	}
	if n := fake.tokens.Load(); n != 2 {
		t.Errorf("esperado 2 emissões de token (reemissão após 401), obtido %d", n)
	}
}

func TestDescobertaDeDatacenter(t *testing.T) {
	fake := novoFakeAcronis([]alertaRaw{exemploAlertaRaw()})
	defer fake.servidor.Close()

	svc := NovoServico([]Conta{{
		Login:        "operador@empresa",
		ClientID:     CLIENT_ID_DEMO,
		ClientSecret: CLIENT_SECRET_DEMO,
	}}, fake.servidor.Client())
	svc.urlDescoberta = fake.servidor.URL // injeta o ponto fixo de descoberta

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("fluxo com descoberta falhou: %v", err)
	}
	if len(resultado.Data) != 1 {
		t.Fatalf("esperado 1 alerta, obtido %d", len(resultado.Data))
	}
	if n := fake.descobertas.Load(); n != 1 {
		t.Errorf("esperado 1 descoberta de datacenter, obtido %d", n)
	}
}

// ----- Multi-conta -----

func TestMultiplasContasAgregamAlertas(t *testing.T) {
	fake := novoFakeAcronis([]alertaRaw{exemploAlertaRaw()})
	defer fake.servidor.Close()

	conta := Conta{
		ServerURL:    fake.servidor.URL,
		ClientID:     CLIENT_ID_DEMO,
		ClientSecret: CLIENT_SECRET_DEMO,
	}
	svc := NovoServico([]Conta{conta, conta}, fake.servidor.Client())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh multi-conta falhou: %v", err)
	}
	if len(resultado.Data) != 2 {
		t.Fatalf("esperado 2 alertas (1 por conta), obtido %d", len(resultado.Data))
	}
	if n := fake.tokens.Load(); n != 2 {
		t.Errorf("esperado 1 token por conta (2), obtido %d", n)
	}
}

func TestContaComCredenciaisRuinsViraFalhaParcial(t *testing.T) {
	fake := novoFakeAcronis([]alertaRaw{exemploAlertaRaw()})
	defer fake.servidor.Close()

	boa := Conta{
		Nome:         "boa",
		ServerURL:    fake.servidor.URL,
		ClientID:     CLIENT_ID_DEMO,
		ClientSecret: CLIENT_SECRET_DEMO,
	}
	ruim := Conta{
		Nome:         "ruim",
		ServerURL:    fake.servidor.URL,
		ClientID:     CLIENT_ID_DEMO,
		ClientSecret: "secret-errado",
	}
	svc := NovoServico([]Conta{boa, ruim}, fake.servidor.Client())

	resultado, err := svc.AtualizarERetornar()
	if err != nil {
		t.Fatalf("refresh não deveria falhar com 1 conta boa: %v", err)
	}
	if len(resultado.Data) != 1 {
		t.Fatalf("esperado 1 alerta (só a conta boa), obtido %d", len(resultado.Data))
	}
	if len(resultado.Falhas) != 1 || resultado.Falhas[0] != "ruim" {
		t.Errorf("esperado falha rotulada 'ruim', obtido %v", resultado.Falhas)
	}
}

// ----- Handlers -----

func TestHandlerRefreshSemContaRetorna400(t *testing.T) {
	h := NovoHandler(NovoServico(nil, nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, ROTA_REFRESH, nil)
	h.refresh(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtido %d", rec.Code)
	}
}

func TestHandlerListarRetornaData(t *testing.T) {
	svc := NovoServico(nil, nil)
	svc.AtivarModoDemo(MocksDemonstracao())
	h := NovoHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, ROTA_LISTAR, nil)
	h.listar(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtido %d", rec.Code)
	}
	var corpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta não é JSON: %v", err)
	}
	if _, ok := corpo["data"]; !ok {
		t.Error("resposta deveria ter campo 'data'")
	}
}

// ----- Mocks (handlers/servidor fake no final) -----

// fakeAcronis simula os três endpoints da API: descoberta, token e alertas.
type fakeAcronis struct {
	servidor         *httptest.Server
	tokens           atomic.Int32
	descobertas      atomic.Int32
	alertas401UmaVez atomic.Bool
	limitEnviado     atomic.Bool
}

func novoFakeAcronis(alertas []alertaRaw) *fakeAcronis {
	f := &fakeAcronis{}
	f.servidor = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case ROTA_DESCOBERTA:
			f.descobertas.Add(1)
			_ = json.NewEncoder(w).Encode(respostaDescoberta{
				Login:     r.URL.Query().Get("login"),
				ServerURL: "http://" + r.Host,
			})
		case ROTA_TOKEN:
			id, secret, ok := r.BasicAuth()
			if !ok || id != CLIENT_ID_DEMO || secret != CLIENT_SECRET_DEMO {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			f.tokens.Add(1)
			_ = json.NewEncoder(w).Encode(respostaToken{
				AccessToken: "tok-acesso",
				TokenType:   "bearer",
				ExpiraEm:    time.Now().Add(time.Hour).Unix(),
				IDToken:     "tok-id",
			})
		case ROTA_ALERTAS:
			if f.alertas401UmaVez.CompareAndSwap(true, false) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !strings.HasPrefix(r.Header.Get("Authorization"), PREFIXO_BEARER) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			f.limitEnviado.Store(r.URL.Query().Get("limit") != "")
			// Envelope realista: inclui "paging" para confirmar que o cliente
			// o ignora (não há cursor de continuação na API).
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items":  alertas,
				"paging": map[string]any{"cursors": map[string]any{}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	return f
}

func exemploAlertaRaw() alertaRaw {
	var bruto alertaRaw
	bruto.ID = "887E19F7-59F5-44DC-9721-64B1C48B4117"
	bruto.Type = "cti.a.p.am.alert.v1.0~a.p.basic.v1.0~vendor.application.malware_detected.v1.0"
	bruto.Category = "cti.a.p.am.category.v1.0~vendor.application.protection.v1.0"
	bruto.Severity = "warning"
	bruto.Details.Title = "Malware Detected"
	bruto.Details.Category = "Protection"
	bruto.Details.Description = "Malicious file quarantined"
	bruto.Details.Fields = map[string]any{"Device ID": "62aedd2b-6556-45d5-a76e-43db475068a7"}
	bruto.CreatedAt = "2023-11-07T14:39:57.556577425Z"
	bruto.Tenant.ID = "286"
	return bruto
}
