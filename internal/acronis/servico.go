// internal/acronis/servico.go
//
// Orquestração da integração Acronis: resolve a URL do datacenter (descoberta
// quando necessário), mantém o access token OAuth2 em cache reemitindo-o
// quando expira (expires_on) ou quando a API responde 401, busca os alertas
// e consolida em cache protegido por mutex. Thread-safe.

package acronis

import (
	"errors"
	"fmt"
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

// ----- Tipo Servico -----

// Servico encapsula a conta Acronis, o access token em cache e o cache de
// alertas em memória.
type Servico struct {
	mu            sync.RWMutex
	conta         Conta
	serverURL     string // datacenter resolvido (de conta.ServerURL ou descoberta)
	token         string
	tokenExpiraEm int64 // expires_on (Unix, segundos)
	cache         []Alerta
	ultimasFalhas []string
	urlDescoberta string
	httpClient    *http.Client
	modoDemo      bool
	mocksDemo     []Alerta
}

// NovoServico constrói o serviço com as credenciais informadas.
// httpCliente == nil usa o cliente padrão.
func NovoServico(conta Conta, httpCliente *http.Client) *Servico {
	if httpCliente == nil {
		httpCliente = httpClientPadrao
	}
	return &Servico{
		conta:         conta,
		serverURL:     strings.TrimRight(conta.ServerURL, "/"),
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

	if err := s.atualizar(); err != nil {
		return ResultadoRefresh{}, err
	}

	alertas, falhas := s.CarregarCache()
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

// Configurado retorna true quando há credenciais suficientes (client_id +
// client_secret + datacenter conhecido ou login para descobri-lo) ou quando
// o serviço está em modo demonstração.
func (s *Servico) Configurado() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.modoDemo {
		return true
	}
	temCredenciais := s.conta.ClientID != "" && s.conta.ClientSecret != ""
	temDestino := s.serverURL != "" || s.conta.Login != ""
	return temCredenciais && temDestino
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

// atualizar resolve o datacenter, garante o token e busca os alertas,
// reemitindo o token uma vez se a API responder 401.
func (s *Servico) atualizar() error {
	serverURL, err := s.garantirServidor()
	if err != nil {
		return err
	}

	brutos, err := s.buscarComToken(serverURL)
	if err != nil {
		return err
	}

	s.gravarCache(converterAlertas(brutos))
	return nil
}

// buscarComToken obtém os alertas com o token em cache; em 401, descarta o
// token, reemite e tenta de novo (uma vez).
func (s *Servico) buscarComToken(serverURL string) ([]alertaRaw, error) {
	token, err := s.garantirToken(serverURL)
	if err != nil {
		return nil, err
	}

	brutos, err := buscarAlertas(s.httpClient, serverURL, token)
	if !errors.Is(err, erroTokenExpirado) {
		return brutos, err
	}

	s.invalidarToken()
	token, err = s.garantirToken(serverURL)
	if err != nil {
		return nil, err
	}
	return buscarAlertas(s.httpClient, serverURL, token)
}

// garantirServidor devolve a URL do datacenter, descobrindo-a pelo login
// quando não foi configurada diretamente. O resultado é memorizado.
func (s *Servico) garantirServidor() (string, error) {
	s.mu.RLock()
	servidor, login, urlDescoberta := s.serverURL, s.conta.Login, s.urlDescoberta
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
	s.serverURL = descoberto
	s.mu.Unlock()
	return descoberto, nil
}

// garantirToken devolve o access token em cache enquanto válido (com margem)
// e o reemite via OAuth2 client_credentials quando necessário. A reemissão
// roda fora do lock (não se segura o mutex durante I/O de rede); sob refreshes
// concorrentes isso pode emitir tokens redundantes — aceitável aqui e
// consistente com o padrão de mspclouds/zabbix, sem corromper estado.
func (s *Servico) garantirToken(serverURL string) (string, error) {
	s.mu.RLock()
	token, expira := s.token, s.tokenExpiraEm
	clientID, clientSecret := s.conta.ClientID, s.conta.ClientSecret
	s.mu.RUnlock()

	if token != "" && time.Now().Unix()+MARGEM_EXPIRACAO_SEGUNDOS < expira {
		return token, nil
	}

	novo, err := obterToken(s.httpClient, serverURL, clientID, clientSecret)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.token = novo.AccessToken
	s.tokenExpiraEm = novo.ExpiraEm
	s.mu.Unlock()
	return novo.AccessToken, nil
}

func (s *Servico) invalidarToken() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = ""
	s.tokenExpiraEm = 0
}

func (s *Servico) gravarCache(alertas []Alerta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = alertas
	s.ultimasFalhas = []string{}
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
