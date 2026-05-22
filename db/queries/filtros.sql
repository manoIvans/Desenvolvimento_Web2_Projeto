-- db/queries/filtros.sql
--
-- Queries do CRUD de filtros. Cada filtro pertence a uma instância Zabbix
-- (relacionamento 1:N) — ListarFiltrosPorInstancia alimenta o endpoint
-- aninhado GET /zabbix/instancias/{id}.

-- name: CriarFiltro :one
INSERT INTO filtros (instancia_id, alvo, valor, evento, host)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListarFiltros :many
SELECT * FROM filtros
ORDER BY id;

-- name: ListarFiltrosPorInstancia :many
SELECT * FROM filtros
WHERE instancia_id = $1
ORDER BY id;

-- name: BuscarFiltro :one
SELECT * FROM filtros
WHERE id = $1;

-- name: AtualizarFiltro :one
UPDATE filtros
SET alvo = $2, valor = $3, evento = $4, host = $5, atualizado_em = now()
WHERE id = $1
RETURNING *;

-- name: RemoverFiltro :execrows
DELETE FROM filtros
WHERE id = $1;
