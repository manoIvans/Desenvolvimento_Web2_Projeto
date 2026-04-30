# SignalHub

API HTTP em Go para consolidação de alertas de monitoramento de múltiplas
fontes (Zabbix e MSP Clouds). Projeto didático da disciplina
**Desenvolvimento Web II (DIM0547)** da UFRN.

Cada sprint amplia o projeto em direção ao stack final (Chi + PostgreSQL +
sqlc + JWT + Docker + CI). Este commit entrega a **Sprint 1**.

## O que esta Sprint entrega

- API HTTP com **Chi** como router
- **5 endpoints** funcionando (ver abaixo), chamando de fato as APIs
  externas do Zabbix (JSON-RPC) e MSP Clouds
- **Middlewares** chi: `RequestID`, `RealIP`, `Logger`, `Recoverer`, `cors`
- Testes automatizados com `net/http/httptest` mockando as APIs externas
- GitHub Actions executando lint (gofmt + go vet) + testes + build
- Graceful shutdown em `SIGINT`/`SIGTERM`
- Logger estruturado com `log/slog`

> **Persistência:** cache em memória protegido por mutex. A Sprint 2
> introduz PostgreSQL + sqlc para armazenar o cache e instâncias configuradas.

## Endpoints

| Método | Rota                  | Descrição                                          |
|--------|-----------------------|----------------------------------------------------|
| GET    | `/healthz`            | Liveness check — processo está vivo                |
| GET    | `/zabbix`             | Cache consolidado de problemas Zabbix              |
| POST   | `/zabbix/refresh`     | Força refresh: chama todas as instâncias Zabbix    |
| GET    | `/mspclouds`          | Cache consolidado de alertas MSP Clouds            |
| POST   | `/mspclouds/refresh`  | Força refresh: chama todas as api_keys MSP         |

### Formato de resposta

`GET /zabbix` e `POST /zabbix/refresh`:

```json
{
  "data": [
    {
      "grupo": "Produção",
      "host": "srv-web-01",
      "evento": "CPU acima de 90%",
      "mensagem": "CPU acima de 90%",
      "prio_label": "High",
      "horario": "2026-04-22T14:30:00Z",
      "source": "zabbix_api",
      "instancia": "Instância 1"
    }
  ],
  "falhas": [],
  "versoes": {"Instância 1": "6.4.0"}
}
```

`GET /mspclouds` e `POST /mspclouds/refresh`:

```json
{
  "data": [
    {
      "client": "ACME Corp",
      "type": "backup_failed",
      "product_keyword": "cloudbackup",
      "message": {"login_name": "acme-srv01", "error": "disk full"}
    }
  ],
  "falhas": []
}
```

## Configuração

Copie o template e preencha com suas credenciais:

```bash
cp configuracoes.exemplo.json configuracoes.json
```

```json
{
    "porta_web": ":8080",
    "zabbix_instancias": [
        {
            "nome": "Instância 1",
            "url": "http://seu-zabbix.exemplo.com/zabbix/api_jsonrpc.php",
            "api_key": "sua-api-key-aqui"
        }
    ],
    "msp_instancias": [
        "sua-msp-api-key-aqui"
    ]
}
```

O arquivo `configuracoes.json` está no `.gitignore` — **nunca suba credenciais
reais pro repositório**. O `configuracoes.exemplo.json` serve de referência.

## Rodando localmente

```bash
# Rodar a API (usa ":8080" por padrão; pode ser sobrescrito via SIGNALHUB_ENDERECO)
go run .

# Testes (sem rede — APIs externas são mockadas via httptest.Server)
go test ./...

# Build
go build -o bin/signalhub .
```

Exemplo de uso (após `go run`):

```bash
# Health check
curl http://localhost:8080/healthz

# Força refresh das instâncias Zabbix (chamada real para a API Zabbix)
curl -X POST http://localhost:8080/zabbix/refresh

# Lê o cache consolidado
curl http://localhost:8080/zabbix

# Lista alertas MSP Clouds
curl -X POST http://localhost:8080/mspclouds/refresh
curl http://localhost:8080/mspclouds
```

## Estrutura do projeto

```
signalhub/
├── main.go                         # entrypoint (config, logger, shutdown)
├── configuracoes.exemplo.json      # template (versionado)
├── configuracoes.json              # credenciais reais (git-ignored)
├── go.mod
├── .github/workflows/ci.yml        # lint + test + build
└── internal/
    ├── config/
    │   └── config.go               # leitura do configuracoes.json
    ├── zabbix/
    │   ├── tipos.go                # Problema + tipos internos da API
    │   ├── cliente.go              # cliente JSON-RPC (moderno + legado)
    │   ├── servico.go              # paralelismo + cache + thread-safety
    │   ├── http.go                 # handlers + /zabbix e /zabbix/refresh
    │   └── zabbix_test.go          # testes com httptest simulando API Zabbix
    ├── mspclouds/
    │   ├── cliente.go              # cliente HTTP para /api/v1/alerts
    │   ├── servico.go              # paralelismo + cache + thread-safety
    │   ├── http.go                 # handlers + /mspclouds e /mspclouds/refresh
    │   └── mspclouds_test.go       # testes com httptest simulando API MSP
    ├── saude/
    │   └── http.go                 # endpoint /healthz
    └── servidor/
        └── servidor.go             # composição do router + graceful shutdown
```

## Arquitetura (Sprint 1)

Cada domínio (`zabbix`, `mspclouds`) expõe um `Servico` com:

- **Cache em memória** (slice protegido por `sync.RWMutex`)
- Método `CarregarCache()` — leitura instantânea (barata)
- Método `AtualizarERetornar()` — dispara chamadas paralelas para todas as
  instâncias configuradas, consolida resultados, grava cache e retorna
- Cliente HTTP **injetável** (`*http.Client`) — permite mockar nas
  chamadas com `httptest.Server`

O roteador Chi é montado em `internal/servidor/servidor.go`, recebendo as
dependências (handlers) via struct `Dependencias`. O entrypoint em
`main.go` (raiz) lê o JSON, instancia os serviços e injeta tudo.

## Roadmap

| Sprint   | Entrega                                                      |
|----------|--------------------------------------------------------------|
| Sprint 1 | API Chi, 5 endpoints reais (Zabbix + MSP), testes, CI        |
| Sprint 2 | PostgreSQL + sqlc + Clean Architecture (service/repository)  |
| Sprint 3 | JWT + middleware de autorização                              |
| Sprint 4 | Docker, docker-compose, observabilidade (healthz/readyz)     |

## Stack

- Go 1.23
- [`go-chi/chi/v5`](https://github.com/go-chi/chi) — router HTTP
- [`go-chi/cors`](https://github.com/go-chi/cors) — middleware CORS
- `log/slog` — logging estruturado (stdlib)
- `net/http/httptest` — testes de handler e mock das APIs externas (stdlib)
