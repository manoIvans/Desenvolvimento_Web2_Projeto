// internal/banco/banco.go
//
// Conexão com o PostgreSQL via pool pgx e aplicação idempotente do schema.
// Expõe o *consultas.Queries gerado pelo sqlc para os domínios da API.

package banco

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"SignalHub/internal/banco/consultas"
)

// ----- Constantes -----

const (
	VAR_AMBIENTE_DSN = "DATABASE_URL"
	DSN_PADRAO       = "postgres://signalhub:signalhub@localhost:5432/signalhub?sslmode=disable"
	TIMEOUT_CONEXAO  = 10 * time.Second
)

// ----- Tipo Banco -----

// Banco encapsula o pool de conexões e as consultas tipadas do sqlc.
type Banco struct {
	pool      *pgxpool.Pool
	Consultas *consultas.Queries
}

// ----- API pública -----

// DSN devolve a string de conexão: variável de ambiente DATABASE_URL
// quando definida, ou o padrão local (compatível com o docker-compose.yml).
func DSN() string {
	if valor := os.Getenv(VAR_AMBIENTE_DSN); valor != "" {
		return valor
	}
	return DSN_PADRAO
}

// Conectar abre o pool, valida a conexão com um ping e aplica o schema.
func Conectar(contexto context.Context, dsn, schemaSQL string) (*Banco, error) {
	ctxConexao, cancelar := context.WithTimeout(contexto, TIMEOUT_CONEXAO)
	defer cancelar()

	pool, err := pgxpool.New(ctxConexao, dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir pool de conexões: %w", err)
	}

	if err := pool.Ping(ctxConexao); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping no banco: %w", err)
	}

	if err := aplicarSchema(ctxConexao, pool, schemaSQL); err != nil {
		pool.Close()
		return nil, err
	}

	return &Banco{
		pool:      pool,
		Consultas: consultas.New(pool),
	}, nil
}

// Fechar encerra o pool de conexões.
func (b *Banco) Fechar() {
	if b.pool == nil {
		return
	}
	b.pool.Close()
}

// ----- Internos -----

// aplicarSchema executa o DDL do schema. O schema é idempotente
// (CREATE TABLE IF NOT EXISTS), então rodar a cada boot é seguro.
func aplicarSchema(contexto context.Context, pool *pgxpool.Pool, schemaSQL string) error {
	if schemaSQL == "" {
		return fmt.Errorf("schema vazio — nada a aplicar")
	}
	if _, err := pool.Exec(contexto, schemaSQL); err != nil {
		return fmt.Errorf("aplicar schema: %w", err)
	}
	return nil
}
