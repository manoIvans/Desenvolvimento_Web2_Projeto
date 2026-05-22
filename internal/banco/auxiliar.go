// internal/banco/auxiliar.go
//
// Auxiliares compartilhados pelos domínios persistidos: classificação dos
// erros do PostgreSQL e formatação de timestamps do pgx.

package banco

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// ----- Constantes -----

// Códigos SQLSTATE do PostgreSQL relevantes para a API.
const (
	CODIGO_VIOLACAO_UNICA = "23505" // unique_violation
	CODIGO_VIOLACAO_FK    = "23503" // foreign_key_violation
)

// ----- Classificação de erros -----

// EhNaoEncontrado indica que a query :one não retornou linha alguma.
func EhNaoEncontrado(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// EhConflito indica violação de restrição UNIQUE (registro duplicado).
func EhConflito(err error) bool {
	return codigoSQLState(err) == CODIGO_VIOLACAO_UNICA
}

// EhViolacaoFK indica violação de chave estrangeira (referência inexistente).
func EhViolacaoFK(err error) bool {
	return codigoSQLState(err) == CODIGO_VIOLACAO_FK
}

// ----- Formatação -----

// FormatarHorario converte um timestamp do pgx em ISO 8601 (UTC).
// Devolve string vazia quando o valor é nulo.
func FormatarHorario(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.UTC().Format(time.RFC3339)
}

// ----- Internos -----

func codigoSQLState(err error) string {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return ""
	}
	return pgErr.Code
}
