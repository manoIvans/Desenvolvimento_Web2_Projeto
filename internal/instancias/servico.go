// internal/instancias/servico.go
//
// Regras de negócio do domínio Instâncias (Zabbix, MSP Clouds e contas
// Acronis): validação no servidor e CRUD persistido via consultas sqlc.
// Inclui a leitura aninhada da instância Zabbix com seus filtros
// (relacionamento 1:N).

package instancias

import (
	"context"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"

	"SignalHub/internal/banco/consultas"
	"SignalHub/internal/filtros"
	"SignalHub/internal/resposta"
)

// ----- Constantes -----

const (
	MAX_TAMANHO_NOME          = 120
	MAX_TAMANHO_URL           = 500
	MAX_TAMANHO_API_KEY       = 500
	MAX_TAMANHO_LOGIN         = 200
	MAX_TAMANHO_CLIENT_ID     = 200
	MAX_TAMANHO_CLIENT_SECRET = 500
)

// ----- Tipo Servico -----

// Servico aplica validação e delega a persistência às consultas sqlc.
type Servico struct {
	consultas consultas.Querier
}

// NovoServico constrói o serviço com o Querier informado.
func NovoServico(q consultas.Querier) *Servico {
	return &Servico{consultas: q}
}

// ----- Zabbix: API pública -----

// ListarZabbix devolve todas as instâncias Zabbix cadastradas.
func (s *Servico) ListarZabbix(contexto context.Context) ([]ZabbixInstancia, error) {
	registros, err := s.consultas.ListarZabbixInstancias(contexto)
	if err != nil {
		return nil, err
	}
	return ConverterListaZabbix(registros), nil
}

// BuscarZabbixComFiltros devolve uma instância Zabbix com seus filtros
// aninhados — endpoint que materializa o relacionamento 1:N.
func (s *Servico) BuscarZabbixComFiltros(contexto context.Context, id int32) (ZabbixInstanciaComFiltros, error) {
	registro, err := s.consultas.BuscarZabbixInstancia(contexto, id)
	if err != nil {
		return ZabbixInstanciaComFiltros{}, err
	}

	registrosFiltros, err := s.consultas.ListarFiltrosPorInstancia(contexto, id)
	if err != nil {
		return ZabbixInstanciaComFiltros{}, err
	}

	return ZabbixInstanciaComFiltros{
		ZabbixInstancia: ConverterZabbix(registro),
		Filtros:         filtros.ConverterLista(registrosFiltros),
	}, nil
}

// CriarZabbix valida a entrada e insere uma nova instância Zabbix.
func (s *Servico) CriarZabbix(contexto context.Context, entrada EntradaZabbix) (ZabbixInstancia, error) {
	entrada = normalizarZabbix(entrada)
	if err := validarZabbix(entrada); err != nil {
		return ZabbixInstancia{}, err
	}

	registro, err := s.consultas.CriarZabbixInstancia(contexto, consultas.CriarZabbixInstanciaParams{
		Nome:   entrada.Nome,
		Url:    entrada.URL,
		ApiKey: entrada.APIKey,
	})
	if err != nil {
		return ZabbixInstancia{}, err
	}
	return ConverterZabbix(registro), nil
}

// AtualizarZabbix valida a entrada e sobrescreve uma instância Zabbix.
func (s *Servico) AtualizarZabbix(contexto context.Context, id int32, entrada EntradaZabbix) (ZabbixInstancia, error) {
	entrada = normalizarZabbix(entrada)
	if err := validarZabbix(entrada); err != nil {
		return ZabbixInstancia{}, err
	}

	registro, err := s.consultas.AtualizarZabbixInstancia(contexto, consultas.AtualizarZabbixInstanciaParams{
		ID:     id,
		Nome:   entrada.Nome,
		Url:    entrada.URL,
		ApiKey: entrada.APIKey,
	})
	if err != nil {
		return ZabbixInstancia{}, err
	}
	return ConverterZabbix(registro), nil
}

// RemoverZabbix apaga uma instância Zabbix (e seus filtros, em cascata).
func (s *Servico) RemoverZabbix(contexto context.Context, id int32) error {
	linhas, err := s.consultas.RemoverZabbixInstancia(contexto, id)
	if err != nil {
		return err
	}
	if linhas == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ----- MSP Clouds: API pública -----

// ListarMsp devolve todas as instâncias MSP Clouds cadastradas.
func (s *Servico) ListarMsp(contexto context.Context) ([]MspInstancia, error) {
	registros, err := s.consultas.ListarMspInstancias(contexto)
	if err != nil {
		return nil, err
	}
	return ConverterListaMsp(registros), nil
}

// BuscarMsp devolve uma instância MSP Clouds pelo id.
func (s *Servico) BuscarMsp(contexto context.Context, id int32) (MspInstancia, error) {
	registro, err := s.consultas.BuscarMspInstancia(contexto, id)
	if err != nil {
		return MspInstancia{}, err
	}
	return ConverterMsp(registro), nil
}

// CriarMsp valida a entrada e insere uma nova instância MSP Clouds.
func (s *Servico) CriarMsp(contexto context.Context, entrada EntradaMsp) (MspInstancia, error) {
	chave := strings.TrimSpace(entrada.APIKey)
	if err := validarApiKey(chave); err != nil {
		return MspInstancia{}, err
	}

	registro, err := s.consultas.CriarMspInstancia(contexto, chave)
	if err != nil {
		return MspInstancia{}, err
	}
	return ConverterMsp(registro), nil
}

// AtualizarMsp valida a entrada e sobrescreve uma instância MSP Clouds.
func (s *Servico) AtualizarMsp(contexto context.Context, id int32, entrada EntradaMsp) (MspInstancia, error) {
	chave := strings.TrimSpace(entrada.APIKey)
	if err := validarApiKey(chave); err != nil {
		return MspInstancia{}, err
	}

	registro, err := s.consultas.AtualizarMspInstancia(contexto, consultas.AtualizarMspInstanciaParams{
		ID:     id,
		ApiKey: chave,
	})
	if err != nil {
		return MspInstancia{}, err
	}
	return ConverterMsp(registro), nil
}

// RemoverMsp apaga uma instância MSP Clouds pelo id.
func (s *Servico) RemoverMsp(contexto context.Context, id int32) error {
	linhas, err := s.consultas.RemoverMspInstancia(contexto, id)
	if err != nil {
		return err
	}
	if linhas == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ----- Acronis: API pública -----

// ListarAcronis devolve todas as contas Acronis cadastradas.
func (s *Servico) ListarAcronis(contexto context.Context) ([]AcronisConta, error) {
	registros, err := s.consultas.ListarAcronisContas(contexto)
	if err != nil {
		return nil, err
	}
	return ConverterListaAcronis(registros), nil
}

// BuscarAcronis devolve uma conta Acronis pelo id.
func (s *Servico) BuscarAcronis(contexto context.Context, id int32) (AcronisConta, error) {
	registro, err := s.consultas.BuscarAcronisConta(contexto, id)
	if err != nil {
		return AcronisConta{}, err
	}
	return ConverterAcronis(registro), nil
}

// CriarAcronis valida a entrada e insere uma nova conta Acronis.
func (s *Servico) CriarAcronis(contexto context.Context, entrada EntradaAcronis) (AcronisConta, error) {
	entrada = normalizarAcronis(entrada)
	if err := validarAcronis(entrada); err != nil {
		return AcronisConta{}, err
	}

	registro, err := s.consultas.CriarAcronisConta(contexto, consultas.CriarAcronisContaParams{
		Nome:         entrada.Nome,
		ServerUrl:    entrada.ServerURL,
		Login:        entrada.Login,
		ClientID:     entrada.ClientID,
		ClientSecret: entrada.ClientSecret,
	})
	if err != nil {
		return AcronisConta{}, err
	}
	return ConverterAcronis(registro), nil
}

// AtualizarAcronis valida a entrada e sobrescreve uma conta Acronis.
func (s *Servico) AtualizarAcronis(contexto context.Context, id int32, entrada EntradaAcronis) (AcronisConta, error) {
	entrada = normalizarAcronis(entrada)
	if err := validarAcronis(entrada); err != nil {
		return AcronisConta{}, err
	}

	registro, err := s.consultas.AtualizarAcronisConta(contexto, consultas.AtualizarAcronisContaParams{
		ID:           id,
		Nome:         entrada.Nome,
		ServerUrl:    entrada.ServerURL,
		Login:        entrada.Login,
		ClientID:     entrada.ClientID,
		ClientSecret: entrada.ClientSecret,
	})
	if err != nil {
		return AcronisConta{}, err
	}
	return ConverterAcronis(registro), nil
}

// RemoverAcronis apaga uma conta Acronis pelo id.
func (s *Servico) RemoverAcronis(contexto context.Context, id int32) error {
	linhas, err := s.consultas.RemoverAcronisConta(contexto, id)
	if err != nil {
		return err
	}
	if linhas == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ----- Validação -----

func validarZabbix(entrada EntradaZabbix) error {
	if len(entrada.Nome) > MAX_TAMANHO_NOME {
		return resposta.Validacao("nome excede o tamanho máximo de 120 caracteres")
	}
	if err := validarURL(entrada.URL); err != nil {
		return err
	}
	return validarApiKey(entrada.APIKey)
}

func validarURL(bruta string) error {
	if bruta == "" {
		return resposta.Validacao("url é obrigatória")
	}
	if len(bruta) > MAX_TAMANHO_URL {
		return resposta.Validacao("url excede o tamanho máximo de 500 caracteres")
	}

	endereco, err := url.ParseRequestURI(bruta)
	if err != nil {
		return resposta.Validacao("url inválida")
	}
	if endereco.Scheme != "http" && endereco.Scheme != "https" {
		return resposta.Validacao("url deve usar o esquema http ou https")
	}
	if endereco.Host == "" {
		return resposta.Validacao("url deve conter um host")
	}
	return nil
}

func validarApiKey(chave string) error {
	if chave == "" {
		return resposta.Validacao("api_key é obrigatória")
	}
	if len(chave) > MAX_TAMANHO_API_KEY {
		return resposta.Validacao("api_key excede o tamanho máximo de 500 caracteres")
	}
	return nil
}

// validarAcronis exige client_id e client_secret e ao menos um destino
// (server_url OU login, já que a URL do datacenter pode ser descoberta a
// partir do login). server_url, quando informado, precisa ser http/https.
func validarAcronis(entrada EntradaAcronis) error {
	if len(entrada.Nome) > MAX_TAMANHO_NOME {
		return resposta.Validacao("nome excede o tamanho máximo de 120 caracteres")
	}
	if entrada.ServerURL == "" && entrada.Login == "" {
		return resposta.Validacao("informe server_url ou login para localizar o datacenter")
	}
	if entrada.ServerURL != "" {
		if err := validarURL(entrada.ServerURL); err != nil {
			return err
		}
	}
	if len(entrada.Login) > MAX_TAMANHO_LOGIN {
		return resposta.Validacao("login excede o tamanho máximo de 200 caracteres")
	}
	if entrada.ClientID == "" {
		return resposta.Validacao("client_id é obrigatório")
	}
	if len(entrada.ClientID) > MAX_TAMANHO_CLIENT_ID {
		return resposta.Validacao("client_id excede o tamanho máximo de 200 caracteres")
	}
	if entrada.ClientSecret == "" {
		return resposta.Validacao("client_secret é obrigatório")
	}
	if len(entrada.ClientSecret) > MAX_TAMANHO_CLIENT_SECRET {
		return resposta.Validacao("client_secret excede o tamanho máximo de 500 caracteres")
	}
	return nil
}

// ----- Utilitários -----

func normalizarZabbix(entrada EntradaZabbix) EntradaZabbix {
	entrada.Nome = strings.TrimSpace(entrada.Nome)
	entrada.URL = strings.TrimSpace(entrada.URL)
	entrada.APIKey = strings.TrimSpace(entrada.APIKey)
	return entrada
}

func normalizarAcronis(entrada EntradaAcronis) EntradaAcronis {
	entrada.Nome = strings.TrimSpace(entrada.Nome)
	entrada.ServerURL = strings.TrimSpace(entrada.ServerURL)
	entrada.Login = strings.TrimSpace(entrada.Login)
	entrada.ClientID = strings.TrimSpace(entrada.ClientID)
	entrada.ClientSecret = strings.TrimSpace(entrada.ClientSecret)
	return entrada
}
