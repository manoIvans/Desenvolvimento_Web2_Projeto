-- db/queries/acronis_contas.sql
--
-- Queries do CRUD de contas Acronis (credenciais OAuth2 client_credentials).
-- Parâmetros posicionais ($1, $2…) — sqlc gera prepared statements.

-- name: CriarAcronisConta :one
INSERT INTO acronis_contas (nome, server_url, login, client_id, client_secret)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListarAcronisContas :many
SELECT * FROM acronis_contas
ORDER BY id;

-- name: BuscarAcronisConta :one
SELECT * FROM acronis_contas
WHERE id = $1;

-- name: AtualizarAcronisConta :one
UPDATE acronis_contas
SET nome = $2, server_url = $3, login = $4, client_id = $5, client_secret = $6, atualizado_em = now()
WHERE id = $1
RETURNING *;

-- name: RemoverAcronisConta :execrows
DELETE FROM acronis_contas
WHERE id = $1;
