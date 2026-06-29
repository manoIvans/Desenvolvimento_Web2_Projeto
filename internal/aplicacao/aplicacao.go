// internal/aplicacao/aplicacao.go
//
// Orquestra o ciclo de vida da API SignalHub: logging, conexão ao banco,
// construção dos serviços, subida do servidor HTTP e shutdown elegante.
// Vive fora do package main para manter o entrypoint da raiz enxuto — o
// main.go só embute o schema (a diretiva //go:embed precisa morar lá) e
// chama Executar.

package aplicacao

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"SignalHub/internal/acronis"
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
)

// ----- Entrypoint da aplicação -----

// Executar roda o ciclo de vida completo. esquemaSQL é o embed.FS de
// db/schema fornecido pelo entrypoint (package main).
func Executar(esquemaSQL embed.FS) {
	configurarLogger()

	bd := conectarBancoOuAbortar(esquemaSQL)
	defer bd.Fechar()

	servicoZabbix, servicoMsp := construirServicos(bd)
	servicoAcronis := construirServicoAcronis(bd)
	servicoAuth := construirServicoAutenticacao()
	aquecerCachesIniciais(servicoZabbix, servicoMsp, servicoAcronis)

	srv := iniciarServidor(resolverEndereco(), bd, servicoZabbix, servicoMsp, servicoAcronis, servicoAuth)
	aguardarShutdown(srv)
}

// ----- Banco -----

func conectarBancoOuAbortar(esquemaSQL embed.FS) *banco.Banco {
	schema, err := carregarSchema(esquemaSQL)
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
func carregarSchema(esquemaSQL embed.FS) (string, error) {
	entradas, err := fs.ReadDir(esquemaSQL, DIRETORIO_SCHEMA)
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
		conteudo, err := esquemaSQL.ReadFile(DIRETORIO_SCHEMA + "/" + nome)
		if err != nil {
			return "", err
		}
		construtor.Write(conteudo)
		construtor.WriteString("\n")
	}
	return construtor.String(), nil
}

// ----- Servidor HTTP -----

func iniciarServidor(endereco string, bd *banco.Banco, servicoZabbix *zabbix.Servico, servicoMsp *mspclouds.Servico, servicoAcronis *acronis.Servico, servicoAuth *autenticacao.Servico) *servidor.Servidor {
	router := servidor.MontarRouter(servidor.Dependencias{
		HandlerZabbix:       zabbix.NovoHandler(servicoZabbix),
		HandlerMsp:          mspclouds.NovoHandler(servicoMsp),
		HandlerAcronis:      acronis.NovoHandler(servicoAcronis),
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
