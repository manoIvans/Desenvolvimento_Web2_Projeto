// internal/filtros/filtros_test.go
//
// Testes do CRUD de filtros usando o QuerierSimulado (sem PostgreSQL real).

package filtros

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"SignalHub/internal/banco/consultas/simulado"
)

// ----- Helpers de teste -----

func montarRouter(servico *Servico) http.Handler {
	r := chi.NewRouter()
	NovoHandler(servico).Rotas(r)
	return r
}

func entradaValida() EntradaFiltro {
	return EntradaFiltro{
		InstanciaID: 1,
		Alvo:        "hosts",
		Host:        "srv-web-01",
	}
}

// ----- Testes do Servico: golden path -----

func TestCriarFiltroComDadosValidos(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	filtro, err := servico.Criar(context.Background(), entradaValida())
	if err != nil {
		t.Fatalf("criar filtro válido falhou: %v", err)
	}
	if filtro.ID == 0 {
		t.Error("filtro criado deveria ter id atribuído")
	}
	if filtro.Alvo != "hosts" || filtro.Host != "srv-web-01" {
		t.Errorf("filtro criado com dados errados: %+v", filtro)
	}
}

func TestListarFiltrosDevolveOsCriados(t *testing.T) {
	servico := NovoServico(simulado.Novo())
	contexto := context.Background()

	_, _ = servico.Criar(contexto, entradaValida())
	_, _ = servico.Criar(contexto, entradaValida())

	lista, err := servico.Listar(contexto)
	if err != nil {
		t.Fatalf("listar falhou: %v", err)
	}
	if len(lista) != 2 {
		t.Errorf("esperado 2 filtros, obtido %d", len(lista))
	}
}

func TestAtualizarFiltroSobrescreveCampos(t *testing.T) {
	servico := NovoServico(simulado.Novo())
	contexto := context.Background()

	criado, _ := servico.Criar(contexto, entradaValida())

	atualizado, err := servico.Atualizar(contexto, criado.ID, EntradaFiltro{
		Alvo:   "eventos",
		Evento: "CPU alta",
	})
	if err != nil {
		t.Fatalf("atualizar falhou: %v", err)
	}
	if atualizado.Alvo != "eventos" || atualizado.Evento != "CPU alta" {
		t.Errorf("atualização não aplicada: %+v", atualizado)
	}
}

func TestRemoverFiltroExistente(t *testing.T) {
	servico := NovoServico(simulado.Novo())
	contexto := context.Background()

	criado, _ := servico.Criar(contexto, entradaValida())

	if err := servico.Remover(contexto, criado.ID); err != nil {
		t.Fatalf("remover falhou: %v", err)
	}
}

// ----- Testes do Servico: bordas e falhas -----

func TestCriarFiltroAlvoInvalido(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	entrada := entradaValida()
	entrada.Alvo = "inexistente"

	if _, err := servico.Criar(context.Background(), entrada); err == nil {
		t.Error("esperado erro de validação para alvo inválido")
	}
}

func TestCriarFiltroSemConteudo(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	entrada := EntradaFiltro{InstanciaID: 1, Alvo: "hosts"}

	if _, err := servico.Criar(context.Background(), entrada); err == nil {
		t.Error("esperado erro de validação para filtro sem valor/evento/host")
	}
}

func TestCriarFiltroSemInstancia(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	entrada := entradaValida()
	entrada.InstanciaID = 0

	if _, err := servico.Criar(context.Background(), entrada); err == nil {
		t.Error("esperado erro de validação quando instancia_id ausente")
	}
}

func TestBuscarFiltroInexistenteRetornaErro(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	if _, err := servico.Buscar(context.Background(), 999); err == nil {
		t.Error("esperado erro ao buscar filtro inexistente")
	}
}

func TestRemoverFiltroInexistenteRetornaErro(t *testing.T) {
	servico := NovoServico(simulado.Novo())

	if err := servico.Remover(context.Background(), 999); err == nil {
		t.Error("esperado erro ao remover filtro inexistente")
	}
}

// ----- Testes do Handler HTTP -----

func TestHandlerCriarFiltroRetorna201(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	corpo := `{"instancia_id":1,"alvo":"hosts","host":"srv-web-01"}`
	req := httptest.NewRequest(http.MethodPost, "/filtros", strings.NewReader(corpo))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("esperado 201, obtido %d (corpo: %s)", rec.Code, rec.Body.String())
	}
}

func TestHandlerCriarFiltroInvalidoRetorna400(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	corpo := `{"instancia_id":1,"alvo":"xpto","host":"srv01"}`
	req := httptest.NewRequest(http.MethodPost, "/filtros", strings.NewReader(corpo))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400 para alvo inválido, obtido %d", rec.Code)
	}
}

func TestHandlerBuscarFiltroInexistenteRetorna404(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	req := httptest.NewRequest(http.MethodGet, "/filtros/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("esperado 404, obtido %d", rec.Code)
	}
}

func TestHandlerListarFiltrosRetornaJSON(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	req := httptest.NewRequest(http.MethodGet, "/filtros", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtido %d", rec.Code)
	}

	var corpo map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta não é JSON: %v", err)
	}
	if _, ok := corpo["data"]; !ok {
		t.Error("resposta deveria conter o campo 'data'")
	}
}

func TestHandlerIDNaoNumericoRetorna400(t *testing.T) {
	router := montarRouter(NovoServico(simulado.Novo()))

	req := httptest.NewRequest(http.MethodGet, "/filtros/abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400 para id não numérico, obtido %d", rec.Code)
	}
}
