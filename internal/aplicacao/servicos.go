// internal/aplicacao/servicos.go
//
// Construção dos serviços de domínio (Zabbix, MSP Clouds, Acronis e
// Autenticação) a partir do banco e do ambiente, e o aquecimento inicial
// dos caches. Separado de aplicacao.go para isolar a montagem dos serviços
// do ciclo de vida do processo.

package aplicacao

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"SignalHub/internal/acronis"
	"SignalHub/internal/autenticacao"
	"SignalHub/internal/banco"
	"SignalHub/internal/mspclouds"
	"SignalHub/internal/zabbix"
)

// ----- Constantes -----

const (
	VAR_AMBIENTE_SENHA_LOGIN  = "SIGNALHUB_SENHA_LOGIN"
	VAR_AMBIENTE_TOKEN_MESTRE = "SIGNALHUB_TOKEN_MESTRE"
	VAR_AMBIENTE_SEGREDO_JWT  = "SIGNALHUB_SEGREDO_JWT"
	VAR_AMBIENTE_TTL_ACESSO   = "SIGNALHUB_TTL_TOKEN_SEGUNDOS"
	VAR_AMBIENTE_TTL_REFRESH  = "SIGNALHUB_TTL_REFRESH_SEGUNDOS"

	VAR_AMBIENTE_ACRONIS_URL           = "SIGNALHUB_ACRONIS_URL"
	VAR_AMBIENTE_ACRONIS_LOGIN         = "SIGNALHUB_ACRONIS_LOGIN"
	VAR_AMBIENTE_ACRONIS_CLIENT_ID     = "SIGNALHUB_ACRONIS_CLIENT_ID"
	VAR_AMBIENTE_ACRONIS_CLIENT_SECRET = "SIGNALHUB_ACRONIS_CLIENT_SECRET"

	SENHA_LOGIN_PADRAO  = "umasenhacriativa"
	TOKEN_MESTRE_PADRAO = "mestre-signalhub-imortal"
	SEGREDO_JWT_PADRAO  = "segredo-jwt-de-desenvolvimento"
	TTL_ACESSO_PADRAO   = time.Hour
	TTL_REFRESH_PADRAO  = 24 * time.Hour
)

// ----- Zabbix e MSP Clouds -----

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

// ----- Acronis -----

// construirServicoAcronis monta o serviço Acronis a partir das contas
// persistidas no banco. Quando o banco está vazio, recorre à conta definida
// por variáveis de ambiente (fallback legado) e, na ausência de ambas, entra
// em modo demonstração com dados mock — como Zabbix e MSP fazem.
func construirServicoAcronis(bd *banco.Banco) *acronis.Servico {
	contexto := context.Background()

	contas := carregarContasAcronis(contexto, bd)
	if len(contas) == 0 {
		if conta, ok := contaAcronisDoAmbiente(); ok {
			contas = []acronis.Conta{conta}
		}
	}

	servicoAcronis := acronis.NovoServico(contas, nil)
	if !servicoAcronis.Configurado() {
		servicoAcronis.AtivarModoDemo(acronis.MocksDemonstracao())
		slog.Warn("nenhuma conta Acronis configurada — modo demonstração ativo")
	}
	return servicoAcronis
}

func carregarContasAcronis(contexto context.Context, bd *banco.Banco) []acronis.Conta {
	registros, err := bd.Consultas.ListarAcronisContas(contexto)
	if err != nil {
		slog.Error("falha ao carregar contas Acronis do banco", "erro", err)
		return nil
	}

	contas := make([]acronis.Conta, 0, len(registros))
	for _, r := range registros {
		contas = append(contas, acronis.Conta{
			Nome:         r.Nome,
			ServerURL:    r.ServerUrl,
			Login:        r.Login,
			ClientID:     r.ClientID,
			ClientSecret: r.ClientSecret,
		})
	}
	return contas
}

// contaAcronisDoAmbiente lê a conta Acronis das variáveis de ambiente.
// Devolve ok=false quando nenhuma das variáveis está definida.
func contaAcronisDoAmbiente() (acronis.Conta, bool) {
	conta := acronis.Conta{
		ServerURL:    os.Getenv(VAR_AMBIENTE_ACRONIS_URL),
		Login:        os.Getenv(VAR_AMBIENTE_ACRONIS_LOGIN),
		ClientID:     os.Getenv(VAR_AMBIENTE_ACRONIS_CLIENT_ID),
		ClientSecret: os.Getenv(VAR_AMBIENTE_ACRONIS_CLIENT_SECRET),
	}
	if conta.ServerURL == "" && conta.Login == "" && conta.ClientID == "" && conta.ClientSecret == "" {
		return acronis.Conta{}, false
	}
	return conta, true
}

// ----- Aquecimento inicial dos caches -----

func aquecerCachesIniciais(servicoZabbix *zabbix.Servico, servicoMsp *mspclouds.Servico, servicoAcronis *acronis.Servico) {
	tarefas := []func(){}
	if servicoZabbix.Configurado() {
		tarefas = append(tarefas, func() {
			aquecer("Zabbix", func() error { _, err := servicoZabbix.AtualizarERetornar(); return err })
		})
	}
	if servicoMsp.Configurado() {
		tarefas = append(tarefas, func() {
			aquecer("MSP Clouds", func() error { _, err := servicoMsp.AtualizarERetornar(); return err })
		})
	}
	if servicoAcronis.Configurado() {
		tarefas = append(tarefas, func() {
			aquecer("Acronis", func() error { _, err := servicoAcronis.AtualizarERetornar(); return err })
		})
	}
	executarEmParalelo(tarefas)
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

// ----- Utilitários -----

// aquecer dispara um refresh inicial de uma fonte e loga o resultado.
func aquecer(fonte string, atualizar func() error) {
	if err := atualizar(); err != nil {
		slog.Warn("refresh inicial falhou", "fonte", fonte, "erro", err)
		return
	}
	slog.Info("refresh inicial concluído", "fonte", fonte)
}

// executarEmParalelo roda as tarefas concorrentemente e aguarda todas.
func executarEmParalelo(tarefas []func()) {
	var wg sync.WaitGroup
	for _, tarefa := range tarefas {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tarefa()
		}()
	}
	wg.Wait()
}

func segredoDoAmbiente(envVar, padrao string) string {
	if valor := os.Getenv(envVar); valor != "" {
		return valor
	}
	slog.Warn("usando valor padrão — defina " + envVar + " para sobrescrever")
	return padrao
}

func valorDoAmbiente(envVar, padrao string) string {
	if valor := os.Getenv(envVar); valor != "" {
		return valor
	}
	return padrao
}

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
