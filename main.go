// main.go
//
// Entrypoint da API SignalHub. Lê configuracoes.json, monta os serviços
// Zabbix/MSP, sobe o servidor HTTP (que também serve o frontend em `/`)
// e aguarda SIGINT/SIGTERM.

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"SignalHub/internal/config"
	"SignalHub/internal/mspclouds"
	"SignalHub/internal/servidor"
	"SignalHub/internal/zabbix"
)

// ----- Constantes -----

const (
	ENDERECO_PADRAO    = ":8080"
	VAR_AMBIENTE_PORTA = "SIGNALHUB_ENDERECO"
	TIMEOUT_SHUTDOWN   = 20 * time.Second
)

const TESTANDO = false

// ----- Entrypoint -----

func main() {
	configurarLogger()

	cfg := lerConfiguracoesOuAbortar()
	endereco := resolverEndereco(cfg)

	servicoZabbix, servicoMsp := construirServicos(cfg)
	aquecerCachesIniciais(servicoZabbix, servicoMsp)

	srv := iniciarServidor(endereco, servicoZabbix, servicoMsp)
	aguardarShutdown(srv, endereco)
}

// ----- Bootstrap -----

func lerConfiguracoesOuAbortar() config.Config {
	cfg, err := config.Ler()
	if err != nil {
		slog.Error("erro ao ler configuracoes.json", "erro", err)
		os.Exit(1)
	}
	slog.Info("configuração carregada",
		"instancias_zabbix", len(cfg.ZabbixInstancias),
		"instancias_msp", len(cfg.MspInstancias),
	)
	return cfg
}

func resolverEndereco(cfg config.Config) string {
	endereco := cfg.PortaWeb
	if endereco == "" {
		endereco = ENDERECO_PADRAO
	}
	if valor := os.Getenv(VAR_AMBIENTE_PORTA); valor != "" {
		endereco = valor
	}
	return endereco
}

func construirServicos(cfg config.Config) (*zabbix.Servico, *mspclouds.Servico) {
	servicoZabbix := zabbix.NovoServico(cfg.ZabbixInstancias, nil)
	servicoMsp := mspclouds.NovoServico(cfg.MspInstancias, "", nil)
	return servicoZabbix, servicoMsp
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

func iniciarServidor(endereco string, servicoZabbix *zabbix.Servico, servicoMsp *mspclouds.Servico) *servidor.Servidor {
	router := servidor.MontarRouter(servidor.Dependencias{
		HandlerZabbix: zabbix.NovoHandler(servicoZabbix),
		HandlerMsp:    mspclouds.NovoHandler(servicoMsp),
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

func aguardarShutdown(srv *servidor.Servidor, endereco string) {
	slog.Info("signalhub iniciado", "endereco", endereco)

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

func configurarLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
