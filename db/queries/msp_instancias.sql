-- db/queries/msp_instancias.sql
--
-- Queries do CRUD de instâncias MSP Clouds (apenas api_key).

-- name: CriarMspInstancia :one
INSERT INTO msp_instancias (api_key)
VALUES ($1)
RETURNING *;

-- name: ListarMspInstancias :many
SELECT * FROM msp_instancias
ORDER BY id;

-- name: BuscarMspInstancia :one
SELECT * FROM msp_instancias
WHERE id = $1;

-- name: AtualizarMspInstancia :one
UPDATE msp_instancias
SET api_key = $2, atualizado_em = now()
WHERE id = $1
RETURNING *;

-- name: RemoverMspInstancia :execrows
DELETE FROM msp_instancias
WHERE id = $1;
