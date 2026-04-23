// internal/zabbix/servico.go
//
// Orquestração: busca em todas as instâncias em paralelo, consolida em cache
// protegido por mutex. Thread-safe.

package zabbix

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"SignalHub/internal/config"
)

// Servico encapsula instâncias configuradas + cache em memória.
type Servico struct {
	mu            sync.RWMutex
	instancias    []config.InstanciaZabbix
	cache         []Problema
	ultimasFalhas []string
	versoes       map[string]string
	httpClient    *http.Client
}

// NovoServico constrói o serviço com as instâncias informadas.
// Se httpCliente == nil, usa o cliente HTTP padrão (útil para injetar mock em testes).
func NovoServico(instancias []config.InstanciaZabbix, httpCliente *http.Client) *Servico {
	if httpCliente == nil {
		httpCliente = httpClientPadrao
	}
	return &Servico{
		instancias: instancias,
		versoes:    map[string]string{},
		httpClient: httpCliente,
	}
}

// Configurado retorna true quando há pelo menos uma instância configurada.
func (s *Servico) Configurado() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.instancias) > 0
}

// CarregarCache devolve uma cópia do cache + falhas do último refresh.
func (s *Servico) CarregarCache() ([]Problema, []string, map[string]string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	problemas := make([]Problema, len(s.cache))
	copy(problemas, s.cache)

	falhas := make([]string, len(s.ultimasFalhas))
	copy(falhas, s.ultimasFalhas)

	versoes := make(map[string]string, len(s.versoes))
	for k, v := range s.versoes {
		versoes[k] = v
	}
	return problemas, falhas, versoes
}

// AtualizarERetornar força um refresh síncrono e devolve o resultado completo.
func (s *Servico) AtualizarERetornar() (ResultadoRefresh, error) {
	if !s.Configurado() {
		return ResultadoRefresh{}, fmt.Errorf("nenhuma instância Zabbix configurada")
	}

	falhas, err := s.atualizar()
	if err != nil {
		return ResultadoRefresh{}, err
	}

	problemas, _, versoes := s.CarregarCache()
	return ResultadoRefresh{
		Data:    problemas,
		Falhas:  falhas,
		Versoes: versoes,
	}, nil
}

// ----- Lógica interna -----

// atualizar busca dados de todas as instâncias em paralelo e atualiza o cache.
// Retorna as URLs das instâncias que falharam.
func (s *Servico) atualizar() ([]string, error) {
	s.mu.RLock()
	instancias := make([]config.InstanciaZabbix, len(s.instancias))
	copy(instancias, s.instancias)
	s.mu.RUnlock()

	if len(instancias) == 0 {
		return nil, nil
	}

	type resultado struct {
		problemas []Problema
		versao    string
		nome      string
		url       string
		erro      error
	}

	canal := make(chan resultado, len(instancias))
	for _, inst := range instancias {
		inst := inst
		go func() {
			problemas, versao, erro := s.buscarDaInstancia(inst)
			canal <- resultado{
				problemas: problemas,
				versao:    versao,
				nome:      inst.Nome,
				url:       inst.URL,
				erro:      erro,
			}
		}()
	}

	var todos []Problema
	novoVersoes := map[string]string{}
	var falhas []string

	for range instancias {
		r := <-canal
		if r.erro != nil {
			falhas = append(falhas, r.url)
			slog.Warn("instância Zabbix falhou", "url", r.url, "erro", r.erro)
			continue
		}
		todos = append(todos, r.problemas...)
		if r.versao != "" && r.nome != "" {
			novoVersoes[r.nome] = r.versao
		}
	}

	if len(todos) == 0 && len(falhas) > 0 {
		return falhas, fmt.Errorf("todas as instâncias Zabbix falharam (%d)", len(falhas))
	}
	if falhas == nil {
		falhas = []string{}
	}

	s.mu.Lock()
	s.cache = todos
	s.ultimasFalhas = falhas
	s.versoes = novoVersoes
	s.mu.Unlock()

	return falhas, nil
}

// buscarDaInstancia faz trigger.get + apiinfo.version em paralelo para uma instância.
func (s *Servico) buscarDaInstancia(inst config.InstanciaZabbix) ([]Problema, string, error) {
	if inst.URL == "" {
		return nil, "", fmt.Errorf("url não informada")
	}
	if inst.APIKey == "" {
		return nil, "", fmt.Errorf("api_key não informada")
	}

	type resultado struct {
		triggers []triggerRaw
		versao   string
		erro     error
		qual     int
	}
	canal := make(chan resultado, 2)

	go func() {
		var triggers []triggerRaw
		erro := chamarApiAdaptado(s.httpClient, inst.URL, inst.APIKey, "trigger.get", map[string]any{
			"output":            []string{"triggerid", "description", "priority", "lastchange", "comments", "url", "opdata"},
			"selectHosts":       []string{"hostid", "host", "name"},
			"selectGroups":      []string{"groupid", "name"},
			"selectTags":        "extend",
			"expandDescription": true,
			"expandComment":     true,
			"monitored":         true,
			"filter":            map[string]any{"value": 1},
			"sortfield":         "lastchange",
			"sortorder":         "DESC",
			"limit":             500,
		}, &triggers)
		canal <- resultado{triggers: triggers, erro: erro, qual: 0}
	}()

	go func() {
		var versao string
		_ = chamarApiLegado(s.httpClient, inst.URL, "", "apiinfo.version", map[string]any{}, &versao)
		canal <- resultado{versao: versao, qual: 1}
	}()

	var triggers []triggerRaw
	var versao string
	for i := 0; i < 2; i++ {
		r := <-canal
		switch r.qual {
		case 0:
			if r.erro != nil {
				return nil, "", r.erro
			}
			triggers = r.triggers
		case 1:
			versao = r.versao
		}
	}

	return converterTriggers(triggers, inst.Nome), versao, nil
}

// converterTriggers transforma o formato bruto da API em []Problema.
func converterTriggers(triggers []triggerRaw, nomeInstancia string) []Problema {
	problemas := make([]Problema, 0, len(triggers))

	for _, t := range triggers {
		problema := Problema{
			Evento:    t.Description,
			Mensagem:  t.Description,
			PrioLabel: prioridadeParaRotulo(t.Priority.String()),
			Horario:   clockParaRFC3339(t.LastChange.String()),
			Source:    "zabbix_api",
			Instancia: nomeInstancia,
			URL:       t.URL,
		}

		if len(t.Hosts) > 0 {
			problema.Host = t.Hosts[0].Name
			for _, h := range t.Hosts {
				problema.Hosts = append(problema.Hosts, h.Name)
			}
		}
		if len(t.Groups) > 0 {
			problema.Grupo = t.Groups[0].Name
			for _, g := range t.Groups {
				problema.Grupos = append(problema.Grupos, g.Name)
			}
		}
		for _, tag := range t.Tags {
			problema.Tags = append(problema.Tags, ProblemaTag{Tag: tag.Tag, Value: tag.Value})
		}
		problemas = append(problemas, problema)
	}
	return problemas
}

func prioridadeParaRotulo(p string) string {
	switch p {
	case "0":
		return "Not classified"
	case "1":
		return "Information"
	case "2":
		return "Warning"
	case "3":
		return "Average"
	case "4":
		return "High"
	case "5":
		return "Disaster"
	}
	return ""
}

func clockParaRFC3339(clock string) string {
	if clock == "" || clock == "0" {
		return ""
	}
	segundos, err := strconv.ParseInt(clock, 10, 64)
	if err != nil {
		return ""
	}
	return time.Unix(segundos, 0).UTC().Format(time.RFC3339)
}
