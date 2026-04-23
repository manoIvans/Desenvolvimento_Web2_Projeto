// cmd/api/main.go
//
// Entrypoint da API SignalHub. Lê configuracoes.json, monta os serviços
// Zabbix/MSP, sobe o servidor HTTP e espera SIGINT/SIGTERM.

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

const (
	ENDERECO_PADRAO    = ":8080"
	VAR_AMBIENTE_PORTA = "SIGNALHUB_ENDERECO"
)

func main() {
	configurarLogger()

	// ----- Config -----
	cfg, err := config.Ler()
	if err != nil {
		slog.Error("erro ao ler configuracoes.json", "erro", err)
		os.Exit(1)
	}
	slog.Info("configuração carregada",
		"instancias_zabbix", len(cfg.ZabbixInstancias),
		"instancias_msp", len(cfg.MspInstancias),
	)

	endereco := cfg.PortaWeb
	if endereco == "" {
		endereco = ENDERECO_PADRAO
	}
	if valor := os.Getenv(VAR_AMBIENTE_PORTA); valor != "" {
		endereco = valor
	}

	// ----- Serviços -----
	servicoZabbix := zabbix.NovoServico(cfg.ZabbixInstancias, nil)
	servicoMsp := mspclouds.NovoServico(cfg.MspInstancias, "", nil)

	// ----- Refresh inicial -----
	var wgRefresh sync.WaitGroup
	if servicoZabbix.Configurado() {
		wgRefresh.Add(1)
		go func() {
			defer wgRefresh.Done()
			if _, err := servicoZabbix.AtualizarERetornar(); err != nil {
				slog.Warn("refresh inicial Zabbix falhou", "erro", err)
			} else {
				slog.Info("refresh inicial Zabbix concluído")
			}
		}()
	}
	if servicoMsp.Configurado() {
		wgRefresh.Add(1)
		go func() {
			defer wgRefresh.Done()
			if _, err := servicoMsp.AtualizarERetornar(); err != nil {
				slog.Warn("refresh inicial MSP Clouds falhou", "erro", err)
			} else {
				slog.Info("refresh inicial MSP Clouds concluído")
			}
		}()
	}
	wgRefresh.Wait()

	// ----- Router -----
	router := servidor.MontarRouter(servidor.Dependencias{
		HandlerZabbix: zabbix.NovoHandler(servicoZabbix),
		HandlerMsp:    mspclouds.NovoHandler(servicoMsp),
	})

	// ----- Servidor -----
	srv := servidor.Novo(endereco, router)

	erroServidor := make(chan error, 1)
	go func() {
		erroServidor <- srv.Iniciar()
	}()

	slog.Info("signalhub iniciado", "endereco", endereco)

	// ----- Shutdown -----
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("sinal recebido, iniciando shutdown", "sinal", sig.String())
	case err := <-erroServidor:
		if err != nil {
			slog.Error("servidor terminou com erro", "erro", err)
			os.Exit(1)
		}
	}

	ctxShutdown, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	if err := srv.Parar(ctxShutdown); err != nil {
		slog.Error("erro no shutdown", "erro", err)
	}
	slog.Info("encerrado")
}

func configurarLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
