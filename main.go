// main.go
//
// Entrypoint da API SignalHub. Conecta ao PostgreSQL, aplica o schema,
// carrega as instâncias Zabbix/MSP persistidas, sobe o servidor HTTP
// (que também serve o frontend em `/`) e aguarda SIGINT/SIGTERM.

package main

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"SignalHub/internal/autenticacao"
	"SignalHub/internal/banco"
	"SignalHub/internal/filtros"
	"SignalHub/internal/instancias"
	"SignalHub/internal/mspclouds"
	"SignalHub/internal/servidor"
	"SignalHub/internal/zabbix"
)

// ----- Constantes -----

const (
	ENDERECO_PADRAO    = ":8080"
	VAR_AMBIENTE_PORTA = "SIGNALHUB_ENDERECO"
	TIMEOUT_SHUTDOWN   = 20 * time.Second
	DIRETORIO_SCHEMA   = "db/schema"

	VAR_AMBIENTE_SENHA_LOGIN  = "SIGNALHUB_SENHA_LOGIN"
	VAR_AMBIENTE_TOKEN_MESTRE = "SIGNALHUB_TOKEN_MESTRE"
	VAR_AMBIENTE_SEGREDO_JWT  = "SIGNALHUB_SEGREDO_JWT"
	VAR_AMBIENTE_TTL_ACESSO   = "SIGNALHUB_TTL_TOKEN_SEGUNDOS"
	VAR_AMBIENTE_TTL_REFRESH  = "SIGNALHUB_TTL_REFRESH_SEGUNDOS"

	SENHA_LOGIN_PADRAO  = "umasenhacriativa"
	TOKEN_MESTRE_PADRAO = "mestre-signalhub-imortal"
	SEGREDO_JWT_PADRAO  = "segredo-jwt-de-desenvolvimento"
	TTL_ACESSO_PADRAO   = time.Hour
	TTL_REFRESH_PADRAO  = 24 * time.Hour
)

//go:embed db/schema
var arquivosSchema embed.FS

// ----- Entrypoint -----

func main() {
	configurarLogger()

	bd := conectarBancoOuAbortar()
	defer bd.Fechar()

	servicoZabbix, servicoMsp := construirServicos(bd)
	servicoAuth := construirServicoAutenticacao()
	aquecerCachesIniciais(servicoZabbix, servicoMsp)

	srv := iniciarServidor(resolverEndereco(), bd, servicoZabbix, servicoMsp, servicoAuth)
	aguardarShutdown(srv)
}

// ----- Banco -----

func conectarBancoOuAbortar() *banco.Banco {
	schema, err := carregarSchema()
	if err != nil {
		slog.Error("falha ao carregar schema embutido", "erro", err)
		os.Exit(1)
	}

	bd, err := banco.Conectar(context.Background(), banco.DSN(), schema)
	if err != nil {
		slog.Error("falha ao conectar ao PostgreSQL", "erro", err,
			"dica", "suba o banco com 'docker compose up -d'")
		os.Exit(1)
	}
	slog.Info("conectado ao PostgreSQL — schema aplicado")
	return bd
}

// carregarSchema lê e concatena os arquivos .sql embutidos de db/schema,
// em ordem alfabética (a numeração dos arquivos define a ordem de aplicação).
func carregarSchema() (string, error) {
	entradas, err := fs.ReadDir(arquivosSchema, DIRETORIO_SCHEMA)
	if err != nil {
		return "", err
	}

	var nomes []string
	for _, entrada := range entradas {
		if entrada.IsDir() || !strings.HasSuffix(entrada.Name(), ".sql") {
			continue
		}
		nomes = append(nomes, entrada.Name())
	}
	sort.Strings(nomes)

	var construtor strings.Builder
	for _, nome := range nomes {
		conteudo, err := arquivosSchema.ReadFile(DIRETORIO_SCHEMA + "/" + nome)
		if err != nil {
			return "", err
		}
		construtor.Write(conteudo)
		construtor.WriteString("\n")
	}
	return construtor.String(), nil
}

// ----- Serviços -----

// construirServicos lê as instâncias persistidas no banco e monta os
// serviços Zabbix/MSP. Quando não há instâncias cadastradas, o serviço
// entra em modo demonstração com dados mock.
func construirServicos(bd *banco.Banco) (*zabbix.Servico, *mspclouds.Servico) {
	contexto := context.Background()

	servicoZabbix := zabbix.NovoServico(carregarInstanciasZabbix(contexto, bd), nil)
	servicoMsp := mspclouds.NovoServico(carregarChavesMsp(contexto, bd), "", nil)

	if !servicoZabbix.Configurado() {
		servicoZabbix.AtivarModoDemo(zabbix.MocksDemonstracao())
		slog.Warn("nenhuma instância Zabbix no banco — modo demonstração ativo")
	}
	if !servicoMsp.Configurado() {
		servicoMsp.AtivarModoDemo(mspclouds.MocksDemonstracao())
		slog.Warn("nenhuma api_key MSP Clouds no banco — modo demonstração ativo")
	}
	return servicoZabbix, servicoMsp
}

func carregarInstanciasZabbix(contexto context.Context, bd *banco.Banco) []zabbix.InstanciaConfig {
	registros, err := bd.Consultas.ListarZabbixInstancias(contexto)
	if err != nil {
		slog.Error("falha ao carregar instâncias Zabbix do banco", "erro", err)
		return nil
	}

	lista := make([]zabbix.InstanciaConfig, 0, len(registros))
	for _, r := range registros {
		lista = append(lista, zabbix.InstanciaConfig{
			Nome:   r.Nome,
			URL:    r.Url,
			APIKey: r.ApiKey,
		})
	}
	return lista
}

func carregarChavesMsp(contexto context.Context, bd *banco.Banco) []string {
	registros, err := bd.Consultas.ListarMspInstancias(contexto)
	if err != nil {
		slog.Error("falha ao carregar instâncias MSP Clouds do banco", "erro", err)
		return nil
	}

	chaves := make([]string, 0, len(registros))
	for _, r := range registros {
		chaves = append(chaves, r.ApiKey)
	}
	return chaves
}

// ----- Refresh inicial -----

func aquecerCachesIniciais(servicoZabbix *zabbix.Servico, servicoMsp *mspclouds.Servico) {
	var wgRefresh sync.WaitGroup

	if servicoZabbix.Configurado() {
		wgRefresh.Add(1)
		go func() {
			defer wgRefresh.Done()
			aquecerZabbix(servicoZabbix)
		}()
	}

	if servicoMsp.Configurado() {
		wgRefresh.Add(1)
		go func() {
			defer wgRefresh.Done()
			aquecerMsp(servicoMsp)
		}()
	}

	wgRefresh.Wait()
}

func aquecerZabbix(servicoZabbix *zabbix.Servico) {
	if _, err := servicoZabbix.AtualizarERetornar(); err != nil {
		slog.Warn("refresh inicial Zabbix falhou", "erro", err)
		return
	}
	slog.Info("refresh inicial Zabbix concluído")
}

func aquecerMsp(servicoMsp *mspclouds.Servico) {
	if _, err := servicoMsp.AtualizarERetornar(); err != nil {
		slog.Warn("refresh inicial MSP Clouds falhou", "erro", err)
		return
	}
	slog.Info("refresh inicial MSP Clouds concluído")
}

// ----- Servidor HTTP -----

func iniciarServidor(endereco string, bd *banco.Banco, servicoZabbix *zabbix.Servico, servicoMsp *mspclouds.Servico, servicoAuth *autenticacao.Servico) *servidor.Servidor {
	router := servidor.MontarRouter(servidor.Dependencias{
		HandlerZabbix:       zabbix.NovoHandler(servicoZabbix),
		HandlerMsp:          mspclouds.NovoHandler(servicoMsp),
		HandlerInstancias:   instancias.NovoHandler(instancias.NovoServico(bd.Consultas)),
		HandlerFiltros:      filtros.NovoHandler(filtros.NovoServico(bd.Consultas)),
		HandlerAutenticacao: autenticacao.NovoHandler(servicoAuth),
		ServicoAutenticacao: servicoAuth,
	})

	srv := servidor.Novo(endereco, router)
	go func() {
		if err := srv.Iniciar(); err != nil {
			slog.Error("servidor terminou com erro", "erro", err)
			os.Exit(1)
		}
	}()
	return srv
}

// ----- Autenticação -----

// construirServicoAutenticacao lê senha de login, token mestre, segredo de
// assinatura JWT e os TTLs de access/refresh do ambiente. Sem env vars, usa
// os defaults — e loga em WARN para deixar claro que valores padrão estão em
// uso.
func construirServicoAutenticacao() *autenticacao.Servico {
	senha := segredoDoAmbiente(VAR_AMBIENTE_SENHA_LOGIN, SENHA_LOGIN_PADRAO)
	segredoJWT := segredoDoAmbiente(VAR_AMBIENTE_SEGREDO_JWT, SEGREDO_JWT_PADRAO)
	tokenMestre := valorDoAmbiente(VAR_AMBIENTE_TOKEN_MESTRE, TOKEN_MESTRE_PADRAO)

	ttlAcesso := ttlDoAmbiente(VAR_AMBIENTE_TTL_ACESSO, TTL_ACESSO_PADRAO)
	ttlRefresh := ttlDoAmbiente(VAR_AMBIENTE_TTL_REFRESH, TTL_REFRESH_PADRAO)

	slog.Info("autenticação ativa", "ttl_acesso", ttlAcesso.String(), "ttl_refresh", ttlRefresh.String())
	return autenticacao.NovoServico(autenticacao.Config{
		SegredoJWT:  []byte(segredoJWT),
		TokenMestre: tokenMestre,
		SenhaLogin:  senha,
		TTLAcesso:   ttlAcesso,
		TTLRefresh:  ttlRefresh,
	})
}

// segredoDoAmbiente devolve a env var quando definida, ou o padrão — caso em
// que loga um WARN para sinalizar que um valor padrão está em uso.
func segredoDoAmbiente(envVar, padrao string) string {
	if valor := os.Getenv(envVar); valor != "" {
		return valor
	}
	slog.Warn("usando valor padrão — defina " + envVar + " para sobrescrever")
	return padrao
}

// valorDoAmbiente devolve a env var quando definida, ou o padrão.
func valorDoAmbiente(envVar, padrao string) string {
	if valor := os.Getenv(envVar); valor != "" {
		return valor
	}
	return padrao
}

// ttlDoAmbiente lê uma duração em segundos da env var, caindo no padrão
// quando ausente ou inválida.
func ttlDoAmbiente(envVar string, padrao time.Duration) time.Duration {
	bruto := os.Getenv(envVar)
	if bruto == "" {
		return padrao
	}
	segundos, err := strconv.Atoi(bruto)
	if err != nil || segundos <= 0 {
		return padrao
	}
	return time.Duration(segundos) * time.Second
}

// ----- Shutdown -----

func aguardarShutdown(srv *servidor.Servidor) {
	slog.Info("signalhub iniciado", "endereco", srv.Endereco())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	slog.Info("sinal recebido, iniciando shutdown", "sinal", sig.String())

	ctxShutdown, cancelar := context.WithTimeout(context.Background(), TIMEOUT_SHUTDOWN)
	defer cancelar()
	if err := srv.Parar(ctxShutdown); err != nil {
		slog.Error("erro no shutdown", "erro", err)
	}
	slog.Info("encerrado")
}

// ----- Helpers -----

func resolverEndereco() string {
	if valor := os.Getenv(VAR_AMBIENTE_PORTA); valor != "" {
		return valor
	}
	return ENDERECO_PADRAO
}

func configurarLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
