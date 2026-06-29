// internal/acronis/servico.go
//
// Orquestração da integração Acronis multi-conta: para cada conta resolve a
// URL do datacenter (descoberta quando necessário), mantém o access token
// OAuth2 em cache reemitindo-o quando expira (expires_on) ou quando a API
// responde 401, busca os alertas e consolida tudo num cache protegido por
// mutex. As contas são consultadas em paralelo. Thread-safe.

package acronis

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ----- Constantes -----

const (
	// Margem antes de expires_on para reemitir o token preventivamente,
	// evitando usar um token que vence durante a requisição.
	MARGEM_EXPIRACAO_SEGUNDOS = 60
	SOURCE_API_ACRONIS        = "acronis_api"
)

// ----- Tipos -----

// contaRuntime guarda uma conta e o estado mutável do seu token/datacenter.
// Todo acesso aos campos mutáveis (serverURL, token, tokenExpiraEm) é feito
// sob o mutex do Servico; conta é imutável após a construção.
type contaRuntime struct {
	conta         Conta
	serverURL     string // datacenter resolvido (de conta.ServerURL ou descoberta)
	token         string
	tokenExpiraEm int64 // expires_on (Unix, segundos)
}

// Servico encapsula as contas Acronis (cada uma com seu token em cache) e o
// cache de alertas consolidado em memória.
type Servico struct {
	mu            sync.RWMutex
	contas        []*contaRuntime
	cache         []Alerta
	ultimasFalhas []string
	urlDescoberta string
	httpClient    *http.Client
	modoDemo      bool
	mocksDemo     []Alerta
}

// NovoServico constrói o serviço com as contas informadas.
// httpCliente == nil usa o cliente padrão.
func NovoServico(contas []Conta, httpCliente *http.Client) *Servico {
	if httpCliente == nil {
		httpCliente = httpClientPadrao
	}
	runtimes := make([]*contaRuntime, 0, len(contas))
	for _, c := range contas {
		runtimes = append(runtimes, &contaRuntime{
			conta:     c,
			serverURL: strings.TrimRight(c.ServerURL, "/"),
		})
	}
	return &Servico{
		contas:        runtimes,
		urlDescoberta: URL_DESCOBERTA_PADRAO,
		httpClient:    httpCliente,
	}
}

// ----- API pública -----

// AtualizarERetornar força um refresh síncrono e devolve o resultado.
// Em modo demo, repõe o cache com os mocks atuais e retorna sucesso.
func (s *Servico) AtualizarERetornar() (ResultadoRefresh, error) {
	if s.emModoDemo() {
		s.repopularMocks()
		return s.respostaDoCache(), nil
	}
	if !s.Configurado() {
		return ResultadoRefresh{}, fmt.Errorf("nenhuma conta Acronis configurada")
	}

	falhas, err := s.atualizar()
	if err != nil {
		return ResultadoRefresh{}, err
	}

	alertas, _ := s.CarregarCache()
	return ResultadoRefresh{Data: alertas, Falhas: falhas}, nil
}

// AtivarModoDemo popula o cache com mocks e marca o serviço como demo.
func (s *Servico) AtivarModoDemo(mocks []Alerta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modoDemo = true
	s.mocksDemo = mocks
	s.cache = mocks
	s.ultimasFalhas = []string{}
}

// CarregarCache devolve uma cópia do cache + falhas do último refresh.
func (s *Servico) CarregarCache() ([]Alerta, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alertas := make([]Alerta, len(s.cache))
	copy(alertas, s.cache)

	falhas := make([]string, len(s.ultimasFalhas))
	copy(falhas, s.ultimasFalhas)
	return alertas, falhas
}

// Configurado retorna true quando há ao menos uma conta utilizável
// (client_id + client_secret + datacenter conhecido ou login para descobri-lo)
// ou quando o serviço está em modo demonstração.
func (s *Servico) Configurado() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.modoDemo {
		return true
	}
	for _, cr := range s.contas {
		if contaUtilizavel(cr.conta) {
			return true
		}
	}
	return false
}

// ----- Modo demo -----

func (s *Servico) emModoDemo() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modoDemo
}

func (s *Servico) repopularMocks() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = s.mocksDemo
	s.ultimasFalhas = []string{}
}

func (s *Servico) respostaDoCache() ResultadoRefresh {
	alertas, falhas := s.CarregarCache()
	return ResultadoRefresh{Data: alertas, Falhas: falhas}
}

// ----- Lógica interna -----

// atualizar busca os alertas de todas as contas em paralelo e consolida o
// cache. Retorna os rótulos das contas que falharam.
func (s *Servico) atualizar() ([]string, error) {
	contas := s.copiarContas()
	if len(contas) == 0 {
		return nil, nil
	}

	canal := dispararBuscas(s, contas)
	todos, falhas := coletarResultados(canal, len(contas))

	if len(todos) == 0 && len(falhas) > 0 {
		return falhas, fmt.Errorf("todas as contas Acronis falharam (%d)", len(falhas))
	}
	if falhas == nil {
		falhas = []string{}
	}

	s.gravarCache(todos, falhas)
	return falhas, nil
}

// resultadoBusca agrega o retorno de uma conta para o coletor.
type resultadoBusca struct {
	alertas []Alerta
	rotulo  string
	erro    error
}

func dispararBuscas(s *Servico, contas []*contaRuntime) <-chan resultadoBusca {
	canal := make(chan resultadoBusca, len(contas))
	for _, cr := range contas {
		cr := cr
		go func() {
			brutos, erro := s.buscarDaConta(cr)
			canal <- resultadoBusca{
				alertas: converterAlertas(brutos),
				rotulo:  rotuloConta(cr.conta),
				erro:    erro,
			}
		}()
	}
	return canal
}

func coletarResultados(canal <-chan resultadoBusca, total int) ([]Alerta, []string) {
	var todos []Alerta
	var falhas []string

	for i := 0; i < total; i++ {
		r := <-canal
		if r.erro != nil {
			falhas = append(falhas, r.rotulo)
			slog.Warn("conta Acronis falhou", "conta", r.rotulo, "erro", r.erro)
			continue
		}
		todos = append(todos, r.alertas...)
	}
	return todos, falhas
}

// buscarDaConta resolve o datacenter da conta, garante o token e busca os
// alertas, reemitindo o token uma vez se a API responder 401.
func (s *Servico) buscarDaConta(cr *contaRuntime) ([]alertaRaw, error) {
	if !contaUtilizavel(cr.conta) {
		return nil, fmt.Errorf("conta sem credenciais ou destino")
	}

	serverURL, err := s.garantirServidor(cr)
	if err != nil {
		return nil, err
	}
	return s.buscarComToken(cr, serverURL)
}

// buscarComToken obtém os alertas com o token em cache; em 401, descarta o
// token, reemite e tenta de novo (uma vez).
func (s *Servico) buscarComToken(cr *contaRuntime, serverURL string) ([]alertaRaw, error) {
	token, err := s.garantirToken(cr, serverURL)
	if err != nil {
		return nil, err
	}

	brutos, err := buscarAlertas(s.httpClient, serverURL, token)
	if !errors.Is(err, erroTokenExpirado) {
		return brutos, err
	}

	s.invalidarToken(cr)
	token, err = s.garantirToken(cr, serverURL)
	if err != nil {
		return nil, err
	}
	return buscarAlertas(s.httpClient, serverURL, token)
}

// garantirServidor devolve a URL do datacenter da conta, descobrindo-a pelo
// login quando não foi configurada diretamente. O resultado é memorizado.
func (s *Servico) garantirServidor(cr *contaRuntime) (string, error) {
	s.mu.RLock()
	servidor, login, urlDescoberta := cr.serverURL, cr.conta.Login, s.urlDescoberta
	s.mu.RUnlock()

	if servidor != "" {
		return servidor, nil
	}
	if login == "" {
		return "", fmt.Errorf("sem ServerURL nem Login para localizar o datacenter Acronis")
	}

	descoberto, err := descobrirServidor(s.httpClient, urlDescoberta, login)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	cr.serverURL = descoberto
	s.mu.Unlock()
	return descoberto, nil
}

// garantirToken devolve o access token da conta em cache enquanto válido (com
// margem) e o reemite via OAuth2 client_credentials quando necessário. A
// reemissão roda fora do lock (não se segura o mutex durante I/O de rede);
// sob refreshes concorrentes isso pode emitir tokens redundantes — aceitável
// aqui e consistente com o padrão de mspclouds/zabbix, sem corromper estado.
func (s *Servico) garantirToken(cr *contaRuntime, serverURL string) (string, error) {
	s.mu.RLock()
	token, expira := cr.token, cr.tokenExpiraEm
	clientID, clientSecret := cr.conta.ClientID, cr.conta.ClientSecret
	s.mu.RUnlock()

	if token != "" && time.Now().Unix()+MARGEM_EXPIRACAO_SEGUNDOS < expira {
		return token, nil
	}

	novo, err := obterToken(s.httpClient, serverURL, clientID, clientSecret)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	cr.token = novo.AccessToken
	cr.tokenExpiraEm = novo.ExpiraEm
	s.mu.Unlock()
	return novo.AccessToken, nil
}

func (s *Servico) invalidarToken(cr *contaRuntime) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cr.token = ""
	cr.tokenExpiraEm = 0
}

// ----- Helpers de estado -----

func (s *Servico) copiarContas() []*contaRuntime {
	s.mu.RLock()
	defer s.mu.RUnlock()
	contas := make([]*contaRuntime, len(s.contas))
	copy(contas, s.contas)
	return contas
}

func (s *Servico) gravarCache(alertas []Alerta, falhas []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = alertas
	s.ultimasFalhas = falhas
}

// contaUtilizavel indica se a conta tem credenciais e um destino (URL direta
// ou login para descoberta). Lê apenas campos imutáveis da Conta.
func contaUtilizavel(c Conta) bool {
	temCredenciais := c.ClientID != "" && c.ClientSecret != ""
	temDestino := c.ServerURL != "" || c.Login != ""
	return temCredenciais && temDestino
}

// rotuloConta devolve um identificador legível da conta para logs e falhas.
func rotuloConta(c Conta) string {
	switch {
	case c.Nome != "":
		return c.Nome
	case c.Login != "":
		return c.Login
	case c.ClientID != "":
		return c.ClientID
	default:
		return c.ServerURL
	}
}

// ----- Conversores -----

func converterAlertas(brutos []alertaRaw) []Alerta {
	alertas := make([]Alerta, 0, len(brutos))
	for _, bruto := range brutos {
		alertas = append(alertas, converterAlerta(bruto))
	}
	return alertas
}

func converterAlerta(bruto alertaRaw) Alerta {
	categoria := bruto.Details.Category
	if categoria == "" {
		categoria = bruto.Category
	}
	return Alerta{
		ID:         bruto.ID,
		Tipo:       bruto.Type,
		Categoria:  categoria,
		Severidade: bruto.Severity,
		Titulo:     bruto.Details.Title,
		Descricao:  bruto.Details.Description,
		Detalhes:   bruto.Details.Fields,
		Horario:    bruto.CreatedAt,
		Tenant:     bruto.Tenant.ID,
		Source:     SOURCE_API_ACRONIS,
	}
}
