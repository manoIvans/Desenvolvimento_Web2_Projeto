-- db/schema/0001_esquema_inicial.sql
--
-- Schema inicial do SignalHub (PostgreSQL). Idempotente (IF NOT EXISTS)
-- para poder ser aplicado com segurança a cada boot da API.
--
-- Relacionamento 1:N: zabbix_instancias (1) --< filtros (N).
-- Um filtro pertence a exatamente uma instância Zabbix; remover a
-- instância remove seus filtros em cascata.

-- ----- Instâncias Zabbix -----
CREATE TABLE IF NOT EXISTS zabbix_instancias (
    id             SERIAL       PRIMARY KEY,
    nome           TEXT         NOT NULL DEFAULT '',
    url            TEXT         NOT NULL UNIQUE,
    api_key        TEXT         NOT NULL,
    criado_em      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    atualizado_em  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ----- Instâncias MSP Clouds -----
CREATE TABLE IF NOT EXISTS msp_instancias (
    id             SERIAL       PRIMARY KEY,
    api_key        TEXT         NOT NULL UNIQUE,
    criado_em      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    atualizado_em  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ----- Filtros (pertencem a uma instância Zabbix) -----
CREATE TABLE IF NOT EXISTS filtros (
    id             SERIAL       PRIMARY KEY,
    instancia_id   INTEGER      NOT NULL REFERENCES zabbix_instancias (id) ON DELETE CASCADE,
    alvo           TEXT         NOT NULL CHECK (alvo IN ('hosts', 'eventos', 'grupos', 'eventos_em_hosts')),
    valor          TEXT         NOT NULL DEFAULT '',
    evento         TEXT         NOT NULL DEFAULT '',
    host           TEXT         NOT NULL DEFAULT '',
    criado_em      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    atualizado_em  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX        IF NOT EXISTS idx_filtros_instancia ON filtros (instancia_id);
CREATE INDEX        IF NOT EXISTS idx_filtros_alvo      ON filtros (alvo);
CREATE UNIQUE INDEX IF NOT EXISTS idx_filtros_unicos    ON filtros (instancia_id, alvo, valor, evento, host);
