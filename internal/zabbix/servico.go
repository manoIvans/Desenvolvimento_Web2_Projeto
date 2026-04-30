// internal/zabbix/servico.go
//
// Orquestração: busca em todas as instâncias em paralelo, consolida em
// cache protegido por mutex. Thread-safe.

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

// ----- Constantes -----

const (
	METODO_TRIGGER_GET     = "trigger.get"
	METODO_APIINFO_VERSION = "apiinfo.version"
	LIMITE_TRIGGERS        = 500
	SOURCE_API_ZABBIX      = "zabbix_api"
)

var ROTULOS_PRIORIDADE = map[string]string{
	"0": "Not classified",
	"1": "Information",
	"2": "Warning",
	"3": "Average",
	"4": "High",
	"5": "Disaster",
}

// ----- Tipo Servico -----

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

// ----- API pública -----

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

// CarregarCache devolve uma cópia do cache + falhas + versões do último refresh.
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

// Configurado retorna true quando há pelo menos uma instância configurada.
func (s *Servico) Configurado() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.instancias) > 0
}

// ----- Lógica interna -----

// atualizar busca dados de todas as instâncias em paralelo e atualiza o cache.
// Retorna as URLs das instâncias que falharam.
func (s *Servico) atualizar() ([]string, error) {
	instancias := s.copiarInstancias()
	if len(instancias) == 0 {
		return nil, nil
	}

	canal := dispararBuscas(s, instancias)
	todos, novoVersoes, falhas := coletarResultados(canal, len(instancias))

	if len(todos) == 0 && len(falhas) > 0 {
		return falhas, fmt.Errorf("todas as instâncias Zabbix falharam (%d)", len(falhas))
	}
	if falhas == nil {
		falhas = []string{}
	}

	s.gravarCache(todos, falhas, novoVersoes)
	return falhas, nil
}

// resultadoBusca agrega o retorno de uma instância para o coletor.
type resultadoBusca struct {
	problemas []Problema
	versao    string
	nome      string
	url       string
	erro      error
}

func dispararBuscas(s *Servico, instancias []config.InstanciaZabbix) <-chan resultadoBusca {
	canal := make(chan resultadoBusca, len(instancias))
	for _, inst := range instancias {
		inst := inst
		go func() {
			problemas, versao, erro := s.buscarDaInstancia(inst)
			canal <- resultadoBusca{
				problemas: problemas,
				versao:    versao,
				nome:      inst.Nome,
				url:       inst.URL,
				erro:      erro,
			}
		}()
	}
	return canal
}

func coletarResultados(canal <-chan resultadoBusca, total int) ([]Problema, map[string]string, []string) {
	var todos []Problema
	novoVersoes := map[string]string{}
	var falhas []string

	for i := 0; i < total; i++ {
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
	return todos, novoVersoes, falhas
}

// buscarDaInstancia faz trigger.get + apiinfo.version em paralelo para uma instância.
func (s *Servico) buscarDaInstancia(inst config.InstanciaZabbix) ([]Problema, string, error) {
	if inst.URL == "" {
		return nil, "", fmt.Errorf("url não informada")
	}
	if inst.APIKey == "" {
		return nil, "", fmt.Errorf("api_key não informada")
	}

	type resultadoParcial struct {
		triggers []triggerRaw
		versao   string
		erro     error
		qual     int
	}
	canal := make(chan resultadoParcial, 2)

	go func() {
		triggers, erro := buscarTriggers(s.httpClient, inst)
		canal <- resultadoParcial{triggers: triggers, erro: erro, qual: 0}
	}()

	go func() {
		versao := buscarVersao(s.httpClient, inst.URL)
		canal <- resultadoParcial{versao: versao, qual: 1}
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

func buscarTriggers(cli *http.Client, inst config.InstanciaZabbix) ([]triggerRaw, error) {
	var triggers []triggerRaw
	erro := chamarApiAdaptado(cli, inst.URL, inst.APIKey, METODO_TRIGGER_GET, parametrosTriggers(), &triggers)
	return triggers, erro
}

func buscarVersao(cli *http.Client, url string) string {
	var versao string
	_ = chamarApiLegado(cli, url, "", METODO_APIINFO_VERSION, map[string]any{}, &versao)
	return versao
}

func parametrosTriggers() map[string]any {
	return map[string]any{
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
		"limit":             LIMITE_TRIGGERS,
	}
}

// converterTriggers transforma o formato bruto da API em []Problema.
func converterTriggers(triggers []triggerRaw, nomeInstancia string) []Problema {
	problemas := make([]Problema, 0, len(triggers))
	for _, t := range triggers {
		problemas = append(problemas, converterTrigger(t, nomeInstancia))
	}
	return problemas
}

func converterTrigger(t triggerRaw, nomeInstancia string) Problema {
	problema := Problema{
		Evento:    t.Description,
		Mensagem:  t.Description,
		PrioLabel: prioridadeParaRotulo(t.Priority.String()),
		Horario:   clockParaRFC3339(t.LastChange.String()),
		Source:    SOURCE_API_ZABBIX,
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
	return problema
}

// ----- Helpers de cache -----

func (s *Servico) copiarInstancias() []config.InstanciaZabbix {
	s.mu.RLock()
	defer s.mu.RUnlock()
	instancias := make([]config.InstanciaZabbix, len(s.instancias))
	copy(instancias, s.instancias)
	return instancias
}

func (s *Servico) gravarCache(problemas []Problema, falhas []string, versoes map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = problemas
	s.ultimasFalhas = falhas
	s.versoes = versoes
}

// ----- Conversores -----

func prioridadeParaRotulo(p string) string {
	if rotulo, ok := ROTULOS_PRIORIDADE[p]; ok {
		return rotulo
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
