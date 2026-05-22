// internal/filtros/tipos.go
//
// DTOs de entrada/saída do domínio Filtros e conversão a partir dos
// registros gerados pelo sqlc.

package filtros

import (
	"SignalHub/internal/banco"
	"SignalHub/internal/banco/consultas"
)

// ----- Tipos -----

// Filtro é o DTO de saída — timestamps já formatados em ISO 8601.
type Filtro struct {
	ID           int32  `json:"id"`
	InstanciaID  int32  `json:"instancia_id"`
	Alvo         string `json:"alvo"`
	Valor        string `json:"valor"`
	Evento       string `json:"evento"`
	Host         string `json:"host"`
	CriadoEm     string `json:"criado_em"`
	AtualizadoEm string `json:"atualizado_em"`
}

// EntradaFiltro é o corpo aceito em POST /filtros e PUT /filtros/{id}.
// InstanciaID é ignorado na atualização (um filtro não troca de instância).
type EntradaFiltro struct {
	InstanciaID int32  `json:"instancia_id"`
	Alvo        string `json:"alvo"`
	Valor       string `json:"valor"`
	Evento      string `json:"evento"`
	Host        string `json:"host"`
}

// ----- Conversão -----

// Converter transforma o registro do sqlc no DTO de saída.
func Converter(f consultas.Filtro) Filtro {
	return Filtro{
		ID:           f.ID,
		InstanciaID:  f.InstanciaID,
		Alvo:         f.Alvo,
		Valor:        f.Valor,
		Evento:       f.Evento,
		Host:         f.Host,
		CriadoEm:     banco.FormatarHorario(f.CriadoEm),
		AtualizadoEm: banco.FormatarHorario(f.AtualizadoEm),
	}
}

// ConverterLista aplica Converter a uma fatia de registros.
func ConverterLista(lista []consultas.Filtro) []Filtro {
	saida := make([]Filtro, 0, len(lista))
	for _, f := range lista {
		saida = append(saida, Converter(f))
	}
	return saida
}
