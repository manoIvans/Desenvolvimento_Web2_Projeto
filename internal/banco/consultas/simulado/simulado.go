// internal/banco/consultas/simulado/simulado.go
//
// Implementação em memória de consultas.Querier para testes — dispensa
// um PostgreSQL real. Replica o essencial do comportamento das queries:
// auto-incremento de id, ordenação por id e pgx.ErrNoRows quando o
// registro não existe.

package simulado

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"SignalHub/internal/banco/consultas"
)

// ----- Tipo QuerierSimulado -----

// QuerierSimulado guarda os registros em mapas e implementa consultas.Querier.
type QuerierSimulado struct {
	zabbix  map[int32]consultas.ZabbixInstancia
	msp     map[int32]consultas.MspInstancia
	acronis map[int32]consultas.AcronisConta
	filtros map[int32]consultas.Filtro

	proxIDZabbix  int32
	proxIDMsp     int32
	proxIDAcronis int32
	proxIDFiltro  int32
}

// Novo constrói um QuerierSimulado vazio.
func Novo() *QuerierSimulado {
	return &QuerierSimulado{
		zabbix:        map[int32]consultas.ZabbixInstancia{},
		msp:           map[int32]consultas.MspInstancia{},
		acronis:       map[int32]consultas.AcronisConta{},
		filtros:       map[int32]consultas.Filtro{},
		proxIDZabbix:  1,
		proxIDMsp:     1,
		proxIDAcronis: 1,
		proxIDFiltro:  1,
	}
}

// ----- Zabbix -----

func (q *QuerierSimulado) CriarZabbixInstancia(_ context.Context, arg consultas.CriarZabbixInstanciaParams) (consultas.ZabbixInstancia, error) {
	registro := consultas.ZabbixInstancia{
		ID:           q.proxIDZabbix,
		Nome:         arg.Nome,
		Url:          arg.Url,
		ApiKey:       arg.ApiKey,
		CriadoEm:     agora(),
		AtualizadoEm: agora(),
	}
	q.zabbix[registro.ID] = registro
	q.proxIDZabbix++
	return registro, nil
}

func (q *QuerierSimulado) ListarZabbixInstancias(_ context.Context) ([]consultas.ZabbixInstancia, error) {
	lista := make([]consultas.ZabbixInstancia, 0, len(q.zabbix))
	for _, registro := range q.zabbix {
		lista = append(lista, registro)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].ID < lista[j].ID })
	return lista, nil
}

func (q *QuerierSimulado) BuscarZabbixInstancia(_ context.Context, id int32) (consultas.ZabbixInstancia, error) {
	registro, ok := q.zabbix[id]
	if !ok {
		return consultas.ZabbixInstancia{}, pgx.ErrNoRows
	}
	return registro, nil
}

func (q *QuerierSimulado) AtualizarZabbixInstancia(_ context.Context, arg consultas.AtualizarZabbixInstanciaParams) (consultas.ZabbixInstancia, error) {
	registro, ok := q.zabbix[arg.ID]
	if !ok {
		return consultas.ZabbixInstancia{}, pgx.ErrNoRows
	}
	registro.Nome = arg.Nome
	registro.Url = arg.Url
	registro.ApiKey = arg.ApiKey
	registro.AtualizadoEm = agora()
	q.zabbix[arg.ID] = registro
	return registro, nil
}

func (q *QuerierSimulado) RemoverZabbixInstancia(_ context.Context, id int32) (int64, error) {
	if _, ok := q.zabbix[id]; !ok {
		return 0, nil
	}
	delete(q.zabbix, id)
	q.removerFiltrosDaInstancia(id)
	return 1, nil
}

// ----- MSP Clouds -----

func (q *QuerierSimulado) CriarMspInstancia(_ context.Context, apiKey string) (consultas.MspInstancia, error) {
	registro := consultas.MspInstancia{
		ID:           q.proxIDMsp,
		ApiKey:       apiKey,
		CriadoEm:     agora(),
		AtualizadoEm: agora(),
	}
	q.msp[registro.ID] = registro
	q.proxIDMsp++
	return registro, nil
}

func (q *QuerierSimulado) ListarMspInstancias(_ context.Context) ([]consultas.MspInstancia, error) {
	lista := make([]consultas.MspInstancia, 0, len(q.msp))
	for _, registro := range q.msp {
		lista = append(lista, registro)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].ID < lista[j].ID })
	return lista, nil
}

func (q *QuerierSimulado) BuscarMspInstancia(_ context.Context, id int32) (consultas.MspInstancia, error) {
	registro, ok := q.msp[id]
	if !ok {
		return consultas.MspInstancia{}, pgx.ErrNoRows
	}
	return registro, nil
}

func (q *QuerierSimulado) AtualizarMspInstancia(_ context.Context, arg consultas.AtualizarMspInstanciaParams) (consultas.MspInstancia, error) {
	registro, ok := q.msp[arg.ID]
	if !ok {
		return consultas.MspInstancia{}, pgx.ErrNoRows
	}
	registro.ApiKey = arg.ApiKey
	registro.AtualizadoEm = agora()
	q.msp[arg.ID] = registro
	return registro, nil
}

func (q *QuerierSimulado) RemoverMspInstancia(_ context.Context, id int32) (int64, error) {
	if _, ok := q.msp[id]; !ok {
		return 0, nil
	}
	delete(q.msp, id)
	return 1, nil
}

// ----- Contas Acronis -----

func (q *QuerierSimulado) CriarAcronisConta(_ context.Context, arg consultas.CriarAcronisContaParams) (consultas.AcronisConta, error) {
	registro := consultas.AcronisConta{
		ID:           q.proxIDAcronis,
		Nome:         arg.Nome,
		ServerUrl:    arg.ServerUrl,
		Login:        arg.Login,
		ClientID:     arg.ClientID,
		ClientSecret: arg.ClientSecret,
		CriadoEm:     agora(),
		AtualizadoEm: agora(),
	}
	q.acronis[registro.ID] = registro
	q.proxIDAcronis++
	return registro, nil
}

func (q *QuerierSimulado) ListarAcronisContas(_ context.Context) ([]consultas.AcronisConta, error) {
	lista := make([]consultas.AcronisConta, 0, len(q.acronis))
	for _, registro := range q.acronis {
		lista = append(lista, registro)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].ID < lista[j].ID })
	return lista, nil
}

func (q *QuerierSimulado) BuscarAcronisConta(_ context.Context, id int32) (consultas.AcronisConta, error) {
	registro, ok := q.acronis[id]
	if !ok {
		return consultas.AcronisConta{}, pgx.ErrNoRows
	}
	return registro, nil
}

func (q *QuerierSimulado) AtualizarAcronisConta(_ context.Context, arg consultas.AtualizarAcronisContaParams) (consultas.AcronisConta, error) {
	registro, ok := q.acronis[arg.ID]
	if !ok {
		return consultas.AcronisConta{}, pgx.ErrNoRows
	}
	registro.Nome = arg.Nome
	registro.ServerUrl = arg.ServerUrl
	registro.Login = arg.Login
	registro.ClientID = arg.ClientID
	registro.ClientSecret = arg.ClientSecret
	registro.AtualizadoEm = agora()
	q.acronis[arg.ID] = registro
	return registro, nil
}

func (q *QuerierSimulado) RemoverAcronisConta(_ context.Context, id int32) (int64, error) {
	if _, ok := q.acronis[id]; !ok {
		return 0, nil
	}
	delete(q.acronis, id)
	return 1, nil
}

// ----- Filtros -----

func (q *QuerierSimulado) CriarFiltro(_ context.Context, arg consultas.CriarFiltroParams) (consultas.Filtro, error) {
	registro := consultas.Filtro{
		ID:           q.proxIDFiltro,
		InstanciaID:  arg.InstanciaID,
		Alvo:         arg.Alvo,
		Valor:        arg.Valor,
		Evento:       arg.Evento,
		Host:         arg.Host,
		CriadoEm:     agora(),
		AtualizadoEm: agora(),
	}
	q.filtros[registro.ID] = registro
	q.proxIDFiltro++
	return registro, nil
}

func (q *QuerierSimulado) ListarFiltros(_ context.Context) ([]consultas.Filtro, error) {
	lista := make([]consultas.Filtro, 0, len(q.filtros))
	for _, registro := range q.filtros {
		lista = append(lista, registro)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].ID < lista[j].ID })
	return lista, nil
}

func (q *QuerierSimulado) ListarFiltrosPorInstancia(_ context.Context, instanciaID int32) ([]consultas.Filtro, error) {
	lista := make([]consultas.Filtro, 0)
	for _, registro := range q.filtros {
		if registro.InstanciaID != instanciaID {
			continue
		}
		lista = append(lista, registro)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].ID < lista[j].ID })
	return lista, nil
}

func (q *QuerierSimulado) BuscarFiltro(_ context.Context, id int32) (consultas.Filtro, error) {
	registro, ok := q.filtros[id]
	if !ok {
		return consultas.Filtro{}, pgx.ErrNoRows
	}
	return registro, nil
}

func (q *QuerierSimulado) AtualizarFiltro(_ context.Context, arg consultas.AtualizarFiltroParams) (consultas.Filtro, error) {
	registro, ok := q.filtros[arg.ID]
	if !ok {
		return consultas.Filtro{}, pgx.ErrNoRows
	}
	registro.Alvo = arg.Alvo
	registro.Valor = arg.Valor
	registro.Evento = arg.Evento
	registro.Host = arg.Host
	registro.AtualizadoEm = agora()
	q.filtros[arg.ID] = registro
	return registro, nil
}

func (q *QuerierSimulado) RemoverFiltro(_ context.Context, id int32) (int64, error) {
	if _, ok := q.filtros[id]; !ok {
		return 0, nil
	}
	delete(q.filtros, id)
	return 1, nil
}

// ----- Internos -----

// removerFiltrosDaInstancia replica o ON DELETE CASCADE da chave estrangeira.
func (q *QuerierSimulado) removerFiltrosDaInstancia(instanciaID int32) {
	for id, registro := range q.filtros {
		if registro.InstanciaID != instanciaID {
			continue
		}
		delete(q.filtros, id)
	}
}

func agora() pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now(), Valid: true}
}
