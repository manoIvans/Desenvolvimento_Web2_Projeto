// internal/mspclouds/servico.go
//
// Orquestração: chama todas as api_keys em paralelo e consolida em cache
// protegido por mutex.

package mspclouds

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

// ----- Constantes -----

const (
	TAMANHO_MIN_CHAVE_VISIVEL  = 6
	REPRESENTACAO_CHAVE_OCULTA = "…"
)

// ----- Tipos -----

// ResultadoRefresh é o que o endpoint /mspclouds/refresh devolve.
type ResultadoRefresh struct {
	Data   []Alerta `json:"data"`
	Falhas []string `json:"falhas,omitempty"`
}

// Servico encapsula as chaves MSP + cache em memória.
type Servico struct {
	mu            sync.RWMutex
	chavesApi     []string
	cache         []Alerta
	ultimasFalhas []string
	baseURL       string
	httpClient    *http.Client
	modoDemo      bool
	mocksDemo     []Alerta
}

// NovoServico constrói o serviço com as api_keys informadas.
// baseURL == "" usa URL_BASE_PADRAO. httpCliente == nil usa o cliente padrão.
func NovoServico(chavesApi []string, baseURL string, httpCliente *http.Client) *Servico {
	if baseURL == "" {
		baseURL = URL_BASE_PADRAO
	}
	if httpCliente == nil {
		httpCliente = httpClientPadrao
	}
	return &Servico{
		chavesApi:  chavesApi,
		baseURL:    baseURL,
		httpClient: httpCliente,
	}
}

// ----- API pública -----

// AtualizarERetornar força refresh síncrono e devolve o resultado.
// Em modo demo, repõe o cache com os mocks atuais e retorna sucesso —
// não há api_keys reais para consultar.
func (s *Servico) AtualizarERetornar() (ResultadoRefresh, error) {
	if s.emModoDemo() {
		s.repopularMocks()
		return s.respostaDoCache(), nil
	}
	if !s.Configurado() {
		return ResultadoRefresh{}, fmt.Errorf("nenhuma api_key MSP Clouds configurada")
	}

	falhas, err := s.atualizar()
	if err != nil {
		return ResultadoRefresh{}, err
	}

	alertas, _ := s.CarregarCache()
	return ResultadoRefresh{Data: alertas, Falhas: falhas}, nil
}

// AtivarModoDemo popula o cache com mocks e marca o serviço como demo.
// Configurado() passa a retornar true e AtualizarERetornar() devolve mocks
// sem fazer chamadas externas.
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

// Configurado retorna true quando há ao menos uma api_key configurada
// ou quando o serviço está em modo demonstração.
func (s *Servico) Configurado() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.chavesApi) > 0 || s.modoDemo
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

func (s *Servico) atualizar() ([]string, error) {
	chaves, baseURL := s.copiarConfig()
	if len(chaves) == 0 {
		return nil, nil
	}

	canal := dispararBuscas(s.httpClient, baseURL, chaves)
	todos, falhas := coletarResultados(canal, len(chaves))

	if len(todos) == 0 && len(falhas) > 0 {
		return falhas, fmt.Errorf("todas as instâncias MSP falharam (%d)", len(falhas))
	}
	if falhas == nil {
		falhas = []string{}
	}

	s.gravarCache(todos, falhas)
	return falhas, nil
}

// resultadoBusca agrega o retorno de uma chave MSP para o coletor.
type resultadoBusca struct {
	alertas []Alerta
	chave   string
	erro    error
}

func dispararBuscas(cli *http.Client, baseURL string, chaves []string) <-chan resultadoBusca {
	canal := make(chan resultadoBusca, len(chaves))
	for _, chave := range chaves {
		chave := chave
		go func() {
			alertas, erro := buscarAlertas(cli, baseURL, chave)
			canal <- resultadoBusca{alertas: alertas, chave: chave, erro: erro}
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
			falhas = append(falhas, mascararChave(r.chave))
			slog.Warn("instância MSP falhou", "chave", mascararChave(r.chave), "erro", r.erro)
			continue
		}
		todos = append(todos, r.alertas...)
	}
	return todos, falhas
}

// ----- Helpers de cache -----

func (s *Servico) copiarConfig() ([]string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chaves := make([]string, len(s.chavesApi))
	copy(chaves, s.chavesApi)
	return chaves, s.baseURL
}

func (s *Servico) gravarCache(alertas []Alerta, falhas []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = alertas
	s.ultimasFalhas = falhas
}

// ----- Utilitários -----

// mascararChave evita expor a api_key completa em logs/respostas.
func mascararChave(chave string) string {
	if len(chave) <= TAMANHO_MIN_CHAVE_VISIVEL {
		return REPRESENTACAO_CHAVE_OCULTA
	}
	return chave[:TAMANHO_MIN_CHAVE_VISIVEL] + REPRESENTACAO_CHAVE_OCULTA
}
