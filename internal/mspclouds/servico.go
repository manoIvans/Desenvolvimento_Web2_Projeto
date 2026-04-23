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

// Configurado retorna true quando há ao menos uma api_key configurada.
func (s *Servico) Configurado() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.chavesApi) > 0
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

// AtualizarERetornar força refresh síncrono e devolve o resultado.
func (s *Servico) AtualizarERetornar() (ResultadoRefresh, error) {
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

// ----- Lógica interna -----

func (s *Servico) atualizar() ([]string, error) {
	s.mu.RLock()
	chaves := make([]string, len(s.chavesApi))
	copy(chaves, s.chavesApi)
	baseURL := s.baseURL
	s.mu.RUnlock()

	if len(chaves) == 0 {
		return nil, nil
	}

	type resultado struct {
		alertas []Alerta
		chave   string
		erro    error
	}

	canal := make(chan resultado, len(chaves))
	for _, chave := range chaves {
		chave := chave
		go func() {
			alertas, erro := buscarAlertas(s.httpClient, baseURL, chave)
			canal <- resultado{alertas: alertas, chave: chave, erro: erro}
		}()
	}

	var todos []Alerta
	var falhas []string

	for range chaves {
		r := <-canal
		if r.erro != nil {
			falhas = append(falhas, mascararChave(r.chave))
			slog.Warn("instância MSP falhou", "chave", mascararChave(r.chave), "erro", r.erro)
			continue
		}
		todos = append(todos, r.alertas...)
	}

	if len(todos) == 0 && len(falhas) > 0 {
		return falhas, fmt.Errorf("todas as instâncias MSP falharam (%d)", len(falhas))
	}
	if falhas == nil {
		falhas = []string{}
	}

	s.mu.Lock()
	s.cache = todos
	s.ultimasFalhas = falhas
	s.mu.Unlock()

	return falhas, nil
}

// mascararChave evita expor a api_key completa em logs/respostas.
func mascararChave(chave string) string {
	if len(chave) <= 6 {
		return "…"
	}
	return chave[:6] + "…"
}
