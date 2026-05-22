-- db/queries/zabbix_instancias.sql
--
-- Queries do CRUD de instâncias Zabbix. Parâmetros posicionais ($1, $2…)
-- — sqlc gera prepared statements, sem concatenação de string.

-- name: CriarZabbixInstancia :one
INSERT INTO zabbix_instancias (nome, url, api_key)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListarZabbixInstancias :many
SELECT * FROM zabbix_instancias
ORDER BY id;

-- name: BuscarZabbixInstancia :one
SELECT * FROM zabbix_instancias
WHERE id = $1;

-- name: AtualizarZabbixInstancia :one
UPDATE zabbix_instancias
SET nome = $2, url = $3, api_key = $4, atualizado_em = now()
WHERE id = $1
RETURNING *;

-- name: RemoverZabbixInstancia :execrows
DELETE FROM zabbix_instancias
WHERE id = $1;
