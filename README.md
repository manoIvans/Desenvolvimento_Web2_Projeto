# SignalHub

API HTTP em Go para consolidação de alertas de monitoramento de múltiplas
fontes (Zabbix, MSP Clouds e Acronis Cyber Protect Cloud). Projeto didático
da disciplina **Desenvolvimento Web II (DIM0547)** da UFRN.

Cada sprint amplia o projeto em direção ao stack final (Chi + PostgreSQL +
sqlc + JWT + Docker + CI). Este commit entrega a **Sprint 3**.

## O que a Sprint 3 entrega

- **Autenticação JWT**: `POST /login` devolve um access token JWT (HS256,
  assinado com a stdlib — sem dependências externas) validado de forma
  stateless no middleware
- **Refresh token com rotação**: `POST /refresh` troca um refresh token
  opaco por um par novo e **invalida o anterior** (uso único), limitando a
  janela de reuso de um token vazado; `POST /logout` revoga o refresh
- **Rotas protegidas** por middleware Bearer — todos os agregadores e CRUDs
- **Correções de segurança OWASP**:
  - *Rate limiting* por IP (token bucket) em `/login`, `/refresh` e `/logout`
    contra força bruta (A07: Identification & Authentication Failures)
  - *Cabeçalhos de segurança* em todas as respostas — `X-Content-Type-Options`,
    `X-Frame-Options`, `Referrer-Policy`, `Content-Security-Policy`, HSTS
    (A05: Security Misconfiguration)
  - *Validação/sanitização de entrada* no servidor + *prepared statements*
    via sqlc, já presentes desde a Sprint 2 (A03: Injection)
- Suíte de testes ampliada (JWT, rotação de refresh, rate limiting,
  cabeçalhos e **teste de integração do roteador completo**)

## O que a Sprint 2 entrega

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

> **Autorização**: exceto `/healthz`, as rotas de autenticação e o frontend
> estático, **todas** as rotas exigem o header `Authorization: Bearer <token>`
> (access token JWT obtido no `/login` ou o token mestre configurado).

### Autenticação (Sprint 3)

| Método | Rota        | Corpo                      | Descrição                              |
|--------|-------------|----------------------------|----------------------------------------|
| POST   | `/login`    | `{"senha"}`                | Devolve `token` (JWT) + `refresh_token`|
| POST   | `/refresh`  | `{"refresh_token"}`        | Novo par de tokens (rotaciona refresh) |
| POST   | `/logout`   | `{"refresh_token"}`        | Revoga o refresh token                 |

Resposta de `/login` e `/refresh`:

```json
{
  "token": "<JWT>",
  "refresh_token": "<token opaco>",
  "expira_em": "2026-06-13T12:00:00Z",
  "tipo": "Bearer"
}
```

### Saúde e cache (Sprint 1)

| Método | Rota                 | Descrição                                       |
|--------|----------------------|-------------------------------------------------|
| GET    | `/healthz`           | Liveness check (público)                        |
| GET    | `/zabbix`            | Cache consolidado de problemas Zabbix           |
| POST   | `/zabbix/refresh`    | Força refresh das instâncias Zabbix             |
| GET    | `/mspclouds`         | Cache consolidado de alertas MSP Clouds         |
| POST   | `/mspclouds/refresh` | Força refresh das api_keys MSP                  |

### Acronis Cyber Protect Cloud

| Método | Rota                | Descrição                                       |
|--------|---------------------|-------------------------------------------------|
| GET    | `/acronis`          | Cache consolidado de alertas Acronis            |
| POST   | `/acronis/refresh`  | Força refresh dos alertas Acronis               |

> A integração Acronis implementa o fluxo real da **Cyber Platform API**:
> descoberta de datacenter (`GET cloud.acronis.com/api/1/accounts`),
> autenticação **OAuth2 client_credentials** (`POST <dc>/api/2/idp/token`,
> token cacheado e reemitido por `expires_on`/401) e listagem de alertas
> (`GET <dc>/api/alert_manager/v1/alerts`). Sem credenciais, roda em **modo
> demonstração** com dados mock — como Zabbix e MSP Clouds.

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

### CRUD de contas Acronis

| Método | Rota                    | Descrição                  |
|--------|-------------------------|----------------------------|
| GET    | `/acronis/contas`       | Lista todas as contas      |
| POST   | `/acronis/contas`       | Cria uma conta             |
| GET    | `/acronis/contas/{id}`  | Busca uma conta            |
| PUT    | `/acronis/contas/{id}`  | Atualiza uma conta         |
| DELETE | `/acronis/contas/{id}`  | Remove uma conta           |

> As contas cadastradas alimentam a integração Acronis no próximo boot da API
> (mesmo modelo de Zabbix/MSP, que leem as instâncias do banco na subida).
> Cada conta exige `client_id`, `client_secret` e ao menos um destino
> (`server_url` **ou** `login` para descoberta do datacenter).

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

### Variáveis de ambiente de autenticação

Todas têm um padrão para rodar em desenvolvimento — a API loga um **WARN**
quando usa o padrão, bastando definir a variável para sobrescrever:

| Variável                          | Padrão       | Descrição                                   |
|-----------------------------------|--------------|---------------------------------------------|
| `SIGNALHUB_SENHA_LOGIN`           | `umasenha…`  | Senha aceita em `POST /login`               |
| `SIGNALHUB_SEGREDO_JWT`           | `troque-…`   | Segredo HMAC que assina os access tokens    |
| `SIGNALHUB_TOKEN_MESTRE`          | `mestre-…`   | Bearer permanente para admin/scripts        |
| `SIGNALHUB_TTL_TOKEN_SEGUNDOS`    | `3600`       | Validade do access token (segundos)         |
| `SIGNALHUB_TTL_REFRESH_SEGUNDOS`  | `86400`      | Validade do refresh token (segundos)        |

### Variáveis de ambiente da integração Acronis

As contas Acronis são preferencialmente cadastradas pelo painel (CRUD em
`/acronis/contas`, persistido no banco). As variáveis abaixo são um **fallback
legado**: usadas só quando não há nenhuma conta no banco. Sem contas no banco
nem variáveis definidas, a integração roda em **modo demonstração** (mock).
Para conectar a uma conta real, informe `CLIENT_ID` + `CLIENT_SECRET` e ou a
URL do datacenter (`URL`) ou o `LOGIN` (para descoberta automática do DC):

| Variável                          | Descrição                                            |
|-----------------------------------|------------------------------------------------------|
| `SIGNALHUB_ACRONIS_URL`           | URL base do datacenter (ex.: `https://eu2-cloud.acronis.com`) |
| `SIGNALHUB_ACRONIS_LOGIN`         | Login para descobrir o datacenter (se `URL` ausente) |
| `SIGNALHUB_ACRONIS_CLIENT_ID`     | `client_id` do API client (OAuth2)                   |
| `SIGNALHUB_ACRONIS_CLIENT_SECRET` | `client_secret` do API client (OAuth2)               |

Exemplo de uso:

```bash
# 1. Faz login e captura o access token JWT
TOKEN=$(curl -s -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"senha":"umasenhacriativa"}' | jq -r .token)

# 2. Cria uma instância Zabbix (rota protegida → exige o Bearer)
curl -X POST http://localhost:8080/zabbix/instancias \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"nome":"Produção","url":"https://zabbix.exemplo.com","api_key":"k1"}'

# 3. Cria um filtro vinculado à instância 1
curl -X POST http://localhost:8080/filtros \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"instancia_id":1,"alvo":"hosts","host":"srv-web-01"}'

# 4. Busca a instância com os filtros aninhados (relacionamento 1:N)
curl http://localhost:8080/zabbix/instancias/1 -H "Authorization: Bearer $TOKEN"
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
    ├── autenticacao/               # JWT + refresh com rotação + middleware
    ├── seguranca/                  # rate limiting + cabeçalhos de segurança
    ├── instancias/                 # CRUD de instâncias Zabbix/MSP e contas Acronis + 1:N
    ├── filtros/                    # CRUD de filtros
    ├── resposta/                   # helpers HTTP (JSON, erros, parâmetros)
    ├── zabbix/                     # integração + cache + /zabbix
    ├── mspclouds/                  # integração + cache + /mspclouds
    ├── acronis/                    # OAuth2 + alertas Acronis + cache + /acronis
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
| Sprint 3 | JWT + refresh com rotação + OWASP (rate limit, headers) + testes ✅ |
| Sprint 4 | Docker, docker-compose, observabilidade (healthz/readyz)     |

## Stack

- Go 1.25
- [`go-chi/chi/v5`](https://github.com/go-chi/chi) — router HTTP
- [`go-chi/cors`](https://github.com/go-chi/cors) — middleware CORS
- [`jackc/pgx/v5`](https://github.com/jackc/pgx) — driver e pool PostgreSQL
- [sqlc](https://sqlc.dev) — geração de acesso a dados tipado a partir de SQL
- `log/slog` — logging estruturado (stdlib)
- `net/http/httptest` — testes de handler (stdlib)
