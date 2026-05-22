// internal/filtros/servico.go
//
// Regras de negócio do domínio Filtros: validação no servidor e CRUD
// persistido via consultas sqlc (prepared statements parametrizados).

package filtros

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"SignalHub/internal/banco/consultas"
	"SignalHub/internal/resposta"
)

// ----- Constantes -----

const MAX_TAMANHO_CAMPO = 200

// ALVOS_VALIDOS espelha o CHECK (alvo IN ...) do schema.
var ALVOS_VALIDOS = map[string]bool{
	"hosts":            true,
	"eventos":          true,
	"grupos":           true,
	"eventos_em_hosts": true,
}

// ----- Tipo Servico -----

// Servico aplica validação e delega a persistência às consultas sqlc.
type Servico struct {
	consultas consultas.Querier
}

// NovoServico constrói o serviço com o Querier informado.
func NovoServico(q consultas.Querier) *Servico {
	return &Servico{consultas: q}
}

// ----- API pública -----

// Listar devolve todos os filtros cadastrados.
func (s *Servico) Listar(contexto context.Context) ([]Filtro, error) {
	registros, err := s.consultas.ListarFiltros(contexto)
	if err != nil {
		return nil, err
	}
	return ConverterLista(registros), nil
}

// Buscar devolve um filtro pelo id.
func (s *Servico) Buscar(contexto context.Context, id int32) (Filtro, error) {
	registro, err := s.consultas.BuscarFiltro(contexto, id)
	if err != nil {
		return Filtro{}, err
	}
	return Converter(registro), nil
}

// Criar valida a entrada e insere um novo filtro vinculado a uma instância.
func (s *Servico) Criar(contexto context.Context, entrada EntradaFiltro) (Filtro, error) {
	entrada = normalizarEntrada(entrada)

	if entrada.InstanciaID <= 0 {
		return Filtro{}, resposta.Validacao("instancia_id é obrigatório e deve ser positivo")
	}
	if err := validarConteudo(entrada); err != nil {
		return Filtro{}, err
	}

	registro, err := s.consultas.CriarFiltro(contexto, consultas.CriarFiltroParams{
		InstanciaID: entrada.InstanciaID,
		Alvo:        entrada.Alvo,
		Valor:       entrada.Valor,
		Evento:      entrada.Evento,
		Host:        entrada.Host,
	})
	if err != nil {
		return Filtro{}, err
	}
	return Converter(registro), nil
}

// Atualizar valida a entrada e sobrescreve os campos de conteúdo de um
// filtro. A instância dona não muda.
func (s *Servico) Atualizar(contexto context.Context, id int32, entrada EntradaFiltro) (Filtro, error) {
	entrada = normalizarEntrada(entrada)

	if err := validarConteudo(entrada); err != nil {
		return Filtro{}, err
	}

	registro, err := s.consultas.AtualizarFiltro(contexto, consultas.AtualizarFiltroParams{
		ID:     id,
		Alvo:   entrada.Alvo,
		Valor:  entrada.Valor,
		Evento: entrada.Evento,
		Host:   entrada.Host,
	})
	if err != nil {
		return Filtro{}, err
	}
	return Converter(registro), nil
}

// Remover apaga um filtro pelo id. Devolve pgx.ErrNoRows se nada foi apagado.
func (s *Servico) Remover(contexto context.Context, id int32) error {
	linhas, err := s.consultas.RemoverFiltro(contexto, id)
	if err != nil {
		return err
	}
	if linhas == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ----- Validação -----

func validarConteudo(entrada EntradaFiltro) error {
	if !ALVOS_VALIDOS[entrada.Alvo] {
		return resposta.Validacao("alvo inválido: use hosts, eventos, grupos ou eventos_em_hosts")
	}
	if entrada.Valor == "" && entrada.Evento == "" && entrada.Host == "" {
		return resposta.Validacao("informe ao menos um de: valor, evento ou host")
	}
	if excedeTamanho(entrada.Valor) || excedeTamanho(entrada.Evento) || excedeTamanho(entrada.Host) {
		return resposta.Validacao("campo excede o tamanho máximo de 200 caracteres")
	}
	return nil
}

// ----- Utilitários -----

func normalizarEntrada(entrada EntradaFiltro) EntradaFiltro {
	entrada.Alvo = strings.TrimSpace(entrada.Alvo)
	entrada.Valor = strings.TrimSpace(entrada.Valor)
	entrada.Evento = strings.TrimSpace(entrada.Evento)
	entrada.Host = strings.TrimSpace(entrada.Host)
	return entrada
}

func excedeTamanho(campo string) bool {
	return len(campo) > MAX_TAMANHO_CAMPO
}
