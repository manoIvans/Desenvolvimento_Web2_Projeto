// internal/zabbix/mock.go
//
// Dados de demonstração usados quando configuracoes.json não é encontrado.
// O entrypoint injeta esses problemas no cache via Servico.AtivarModoDemo,
// e o GET /zabbix passa a retorná-los como se viessem de uma instância real.

package zabbix

import "time"

// ----- Constantes -----

const (
	NOME_INSTANCIA_DEMO = "Demo"
	VERSAO_DEMO         = "demo"
	SOURCE_MOCK_ZABBIX  = "zabbix_mock"
)

// ----- API pública -----

// MocksDemonstracao devolve uma lista fixa de problemas para o modo demo.
// Os horários são calculados em relação a `agora` para parecerem recentes.
func MocksDemonstracao() []Problema {
	agora := time.Now().UTC()
	return []Problema{
		montarMock(agora.Add(-12*time.Minute), "Disaster", "PC do Ivan", "Producao",
			"Uso de GPU acima de 95% por mais de 120 minutos (socorro)"),
		montarMock(agora.Add(-25*time.Minute), "High", "HD do Monitor flex (economical) edition do Ivan", "Producao",
			"Espaco em disco /var abaixo de 5%"),
		montarMock(agora.Add(-38*time.Minute), "Disaster", "Internet ta casa do Donk", "Producao",
			"Servico Brisanet parou de responder"),
		montarMock(agora.Add(-1*time.Hour), "Average", "Geladeira SmartFit Pro Max do Xaxá - Ultra Deluxe definitive especial edition 8k", "Infra",
			"Temperatura do congelador acima de 40 graus celcius"),
		montarMock(agora.Add(-2*time.Hour), "Information", "Camera do óculos do donk (está lá)", "Infra",
			"Tarefa de backup noturna concluida com avisos de violação de direitos eridianos"),
		montarMock(agora.Add(-3*time.Hour), "High", "Malware do patinho quack quack", "Servicos",
			"Taxa de cópia de arquivos abaixo de 500GB/min"),
	}
}

// VersoesDemonstracao devolve o map "instancia → versão" exibido no modo demo.
func VersoesDemonstracao() map[string]string {
	return map[string]string{NOME_INSTANCIA_DEMO: VERSAO_DEMO}
}

// ----- Internos -----

func montarMock(horario time.Time, prio, host, grupo, evento string) Problema {
	return Problema{
		Grupo:     grupo,
		Host:      host,
		Hosts:     []string{host},
		Grupos:    []string{grupo},
		Evento:    evento,
		Mensagem:  evento,
		PrioLabel: prio,
		Horario:   horario.Format(time.RFC3339),
		Source:    SOURCE_MOCK_ZABBIX,
		Instancia: NOME_INSTANCIA_DEMO,
	}
}
