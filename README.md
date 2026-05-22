# SignalHub

API HTTP em Go para consolidação de alertas de monitoramento de múltiplas
fontes (Zabbix e MSP Clouds). Projeto didático da disciplina
**Desenvolvimento Web II (DIM0547)** da UFRN.

Cada sprint amplia o projeto em direção ao stack final (Chi + PostgreSQL +
sqlc + JWT + Docker + CI). Este commit entrega a **Sprint 2**.

## O que esta Sprint entrega

- Persistência em **PostgreSQL** via pool de conexões `pgx/v5`
- Acesso a dados gerado por **sqlc** — queries em SQL puro, código Go
  tipado, prepared statements (sem concatenação de string)
- **CRUD completo** persistido em três entidades: instâncias Zabbix,
  instâncias MSP Clouds e filtros
- **Relacionamento 1:N** modelado: `zabbix_instancias (1) ──< filtros (N)`,
  com endpoint que entrega os filtros aninhados na instância
- Schema SQL versionado em [`db/schema/`](db/schema), aplicado de forma
  idempotente no boot da API
- `docker-compose.yml` para subir o PostgreSQL local
- Testes automatizados com um `Querier` em memória (sem depender de banco)
- GitHub Actions: lint (gofmt + go vet) + `sqlc diff` + testes + build

> A Sprint 1 mantinha as instâncias em `configuracoes.json` e o cache em
> memória. A Sprint 2 **substitui o JSON pelo banco**: as instâncias e os
> filtros passam a viver no PostgreSQL, com CRUD via API.

## Modelagem

```
zabbix_instancias (1) ──< filtros (N)        msp_instancias
─────────────────────────────────────        ──────────────
id            SERIAL  PK                      id        SERIAL PK
nome          TEXT                            api_key   TEXT UNIQUE
url           TEXT    UNIQUE                   criado_em / atualizado_em
api_key       TEXT
criado_em / atualizado_em

filtros
───────
id            SERIAL  PK
instancia_id  INTEGER FK → zabbix_instancias(id) ON DELETE CASCADE
alvo          TEXT    CHECK (hosts | eventos | grupos | eventos_em_hosts)
valor / evento / host   TEXT
criado_em / atualizado_em
```

Remover uma instância Zabbix remove seus filtros em cascata.

## Endpoints

### Saúde e cache (Sprint 1)

| Método | Rota                 | Descrição                                       |
|--------|----------------------|-------------------------------------------------|
| GET    | `/healthz`           | Liveness check                                  |
| GET    | `/zabbix`            | Cache consolidado de problemas Zabbix           |
| POST   | `/zabbix/refresh`    | Força refresh das instâncias Zabbix             |
| GET    | `/mspclouds`         | Cache consolidado de alertas MSP Clouds         |
| POST   | `/mspclouds/refresh` | Força refresh das api_keys MSP                  |

### CRUD de instâncias Zabbix (Sprint 2)

| Método | Rota                      | Descrição                                  |
|--------|---------------------------|--------------------------------------------|
| GET    | `/zabbix/instancias`      | Lista todas as instâncias                  |
| POST   | `/zabbix/instancias`      | Cria uma instância                         |
| GET    | `/zabbix/instancias/{id}` | Busca uma instância **com seus filtros**   |
| PUT    | `/zabbix/instancias/{id}` | Atualiza uma instância                     |
| DELETE | `/zabbix/instancias/{id}` | Remove uma instância (e seus filtros)      |

### CRUD de instâncias MSP Clouds (Sprint 2)

| Método | Rota                         | Descrição                  |
|--------|------------------------------|----------------------------|
| GET    | `/mspclouds/instancias`      | Lista todas as instâncias  |
| POST   | `/mspclouds/instancias`      | Cria uma instância         |
| GET    | `/mspclouds/instancias/{id}` | Busca uma instância        |
| PUT    | `/mspclouds/instancias/{id}` | Atualiza uma instância     |
| DELETE | `/mspclouds/instancias/{id}` | Remove uma instância       |

### CRUD de filtros (Sprint 2)

| Método | Rota             | Descrição                  |
|--------|------------------|----------------------------|
| GET    | `/filtros`       | Lista todos os filtros     |
| POST   | `/filtros`       | Cria um filtro             |
| GET    | `/filtros/{id}`  | Busca um filtro            |
| PUT    | `/filtros/{id}`  | Atualiza um filtro         |
| DELETE | `/filtros/{id}`  | Remove um filtro           |

### Exemplo: endpoint 1:N aninhado

`GET /zabbix/instancias/1`:

```json
{
  "id": 1,
  "nome": "Produção",
  "url": "https://zabbix.exemplo.com",
  "api_key": "sua-api-key",
  "criado_em": "2026-05-16T12:00:00Z",
  "atualizado_em": "2026-05-16T12:00:00Z",
  "filtros": [
    { "id": 1, "instancia_id": 1, "alvo": "hosts", "host": "srv-web-01" },
    { "id": 2, "instancia_id": 1, "alvo": "eventos", "evento": "disco cheio" }
  ]
}
```

## Pré-requisitos

- Go 1.25+
- Docker (para o PostgreSQL via `docker-compose.yml`)

## Rodando localmente

```bash
# 1. Sobe o PostgreSQL (usuário/senha/db = signalhub)
docker compose up -d

# 2. Roda a API — conecta ao banco e aplica o schema automaticamente
go run .

# 3. Testes (não precisam de banco — usam um Querier em memória)
go test ./...

# 4. Build
go build -o bin/signalhub .
```

A string de conexão é resolvida da variável `DATABASE_URL`; se ausente,
usa o padrão local `postgres://signalhub:signalhub@localhost:5432/signalhub`.
O endereço de escuta padrão é `:8080`, sobrescrevível por `SIGNALHUB_ENDERECO`.

Exemplo de uso:

```bash
# Cria uma instância Zabbix
curl -X POST http://localhost:8080/zabbix/instancias \
  -H 'Content-Type: application/json' \
  -d '{"nome":"Produção","url":"https://zabbix.exemplo.com","api_key":"k1"}'

# Cria um filtro vinculado à instância 1
curl -X POST http://localhost:8080/filtros \
  -H 'Content-Type: application/json' \
  -d '{"instancia_id":1,"alvo":"hosts","host":"srv-web-01"}'

# Busca a instância com os filtros aninhados (relacionamento 1:N)
curl http://localhost:8080/zabbix/instancias/1
```

## sqlc

O acesso a dados é gerado por [sqlc](https://sqlc.dev): as queries vivem
em [`db/queries/`](db/queries) como SQL puro, e o código Go tipado é
gerado em `internal/banco/consultas/`. Para regenerar após alterar
queries ou schema:

```bash
sqlc generate
```

## Estrutura do projeto

```
signalhub/
├── main.go                         # entrypoint (banco, logger, shutdown)
├── docker-compose.yml              # PostgreSQL local
├── sqlc.yaml                       # configuração do sqlc
├── db/
│   ├── schema/                     # schema SQL versionado (aplicado no boot)
│   └── queries/                    # queries SQL puras consumidas pelo sqlc
├── .github/workflows/ci.yml        # lint + sqlc diff + test + build
└── internal/
    ├── banco/
    │   ├── banco.go                # pool pgx + aplicação do schema
    │   ├── auxiliar.go             # classificação de erros + formatação
    │   └── consultas/              # código gerado pelo sqlc
    │       └── simulado/           # Querier em memória para testes
    ├── instancias/                 # CRUD de instâncias Zabbix e MSP + 1:N
    ├── filtros/                    # CRUD de filtros
    ├── resposta/                   # helpers HTTP (JSON, erros, parâmetros)
    ├── zabbix/                     # integração + cache + /zabbix
    ├── mspclouds/                  # integração + cache + /mspclouds
    ├── saude/                      # endpoint /healthz
    ├── frontend/                   # serve a pasta frontend/ em /
    └── servidor/                   # composição do router + graceful shutdown
```

## Arquitetura

Cada domínio persistido (`instancias`, `filtros`) tem:

- **`Servico`** — validação no servidor (tipo, formato, tamanho) e
  orquestração das consultas
- **`Handler`** — endpoints HTTP, tradução de erros em status (400/404/409/500)
- **`consultas.Querier`** — interface gerada pelo sqlc; em produção é o
  pool pgx, nos testes é o `simulado.QuerierSimulado` em memória

O roteador Chi é montado em `internal/servidor/servidor.go`, recebendo os
handlers via struct `Dependencias`. O entrypoint `main.go` conecta ao
banco, aplica o schema, carrega as instâncias persistidas e injeta tudo.

## Roadmap

| Sprint   | Entrega                                                      |
|----------|--------------------------------------------------------------|
| Sprint 1 | API Chi, 5 endpoints reais (Zabbix + MSP), testes, CI        |
| Sprint 2 | PostgreSQL + sqlc, CRUD persistido, relacionamento 1:N       |
| Sprint 3 | JWT + middleware de autorização + testes de integração      |
| Sprint 4 | Docker, docker-compose, observabilidade (healthz/readyz)     |

## Stack

- Go 1.25
- [`go-chi/chi/v5`](https://github.com/go-chi/chi) — router HTTP
- [`go-chi/cors`](https://github.com/go-chi/cors) — middleware CORS
- [`jackc/pgx/v5`](https://github.com/jackc/pgx) — driver e pool PostgreSQL
- [sqlc](https://sqlc.dev) — geração de acesso a dados tipado a partir de SQL
- `log/slog` — logging estruturado (stdlib)
- `net/http/httptest` — testes de handler (stdlib)
