// internal/instancias/tipos.go
//
// DTOs de entrada/saída do domínio Instâncias (Zabbix e MSP Clouds) e
// conversão a partir dos registros gerados pelo sqlc.

package instancias

import (
	"SignalHub/internal/banco"
	"SignalHub/internal/banco/consultas"
	"SignalHub/internal/filtros"
)

// ----- Tipos de saída -----

// ZabbixInstancia é o DTO de saída de uma instância Zabbix.
type ZabbixInstancia struct {
	ID           int32  `json:"id"`
	Nome         string `json:"nome"`
	URL          string `json:"url"`
	APIKey       string `json:"api_key"`
	CriadoEm     string `json:"criado_em"`
	AtualizadoEm string `json:"atualizado_em"`
}

// ZabbixInstanciaComFiltros é a resposta aninhada do relacionamento 1:N —
// a instância e os filtros que pertencem a ela.
type ZabbixInstanciaComFiltros struct {
	ZabbixInstancia
	Filtros []filtros.Filtro `json:"filtros"`
}

// MspInstancia é o DTO de saída de uma instância MSP Clouds.
type MspInstancia struct {
	ID           int32  `json:"id"`
	APIKey       string `json:"api_key"`
	CriadoEm     string `json:"criado_em"`
	AtualizadoEm string `json:"atualizado_em"`
}

// ----- Tipos de entrada -----

// EntradaZabbix é o corpo aceito em POST/PUT de instâncias Zabbix.
type EntradaZabbix struct {
	Nome   string `json:"nome"`
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// EntradaMsp é o corpo aceito em POST/PUT de instâncias MSP Clouds.
type EntradaMsp struct {
	APIKey string `json:"api_key"`
}

// ----- Conversão -----

// ConverterZabbix transforma o registro do sqlc no DTO de saída.
func ConverterZabbix(z consultas.ZabbixInstancia) ZabbixInstancia {
	return ZabbixInstancia{
		ID:           z.ID,
		Nome:         z.Nome,
		URL:          z.Url,
		APIKey:       z.ApiKey,
		CriadoEm:     banco.FormatarHorario(z.CriadoEm),
		AtualizadoEm: banco.FormatarHorario(z.AtualizadoEm),
	}
}

// ConverterListaZabbix aplica ConverterZabbix a uma fatia de registros.
func ConverterListaZabbix(lista []consultas.ZabbixInstancia) []ZabbixInstancia {
	saida := make([]ZabbixInstancia, 0, len(lista))
	for _, z := range lista {
		saida = append(saida, ConverterZabbix(z))
	}
	return saida
}

// ConverterMsp transforma o registro do sqlc no DTO de saída.
func ConverterMsp(m consultas.MspInstancia) MspInstancia {
	return MspInstancia{
		ID:           m.ID,
		APIKey:       m.ApiKey,
		CriadoEm:     banco.FormatarHorario(m.CriadoEm),
		AtualizadoEm: banco.FormatarHorario(m.AtualizadoEm),
	}
}

// ConverterListaMsp aplica ConverterMsp a uma fatia de registros.
func ConverterListaMsp(lista []consultas.MspInstancia) []MspInstancia {
	saida := make([]MspInstancia, 0, len(lista))
	for _, m := range lista {
		saida = append(saida, ConverterMsp(m))
	}
	return saida
}
