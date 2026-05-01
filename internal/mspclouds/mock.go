// internal/mspclouds/mock.go
//
// Dados de demonstração usados quando configuracoes.json não é encontrado.
// O entrypoint injeta esses alertas no cache via Servico.AtivarModoDemo,
// e o GET /mspclouds passa a retorná-los como se viessem de api_keys reais.

package mspclouds

import "time"

// ----- API pública -----

// MocksDemonstracao devolve uma lista fixa de alertas para o modo demo.
// Os horários são calculados em relação a `agora` para parecerem recentes.
func MocksDemonstracao() []Alerta {
	agora := time.Now().UTC()
	return []Alerta{
		montarMock(agora.Add(-8*time.Minute), "Empresa Farma Aura", "backup_failed", "cloudbackup",
			"Falha no backup diario do servidor de arquivos"),
		montarMock(agora.Add(-22*time.Minute), "Empresa Betinha corp", "connectivity_lost", "msp_n_central",
			"Perda de conexao com link :( "),
		montarMock(agora.Add(-45*time.Minute), "Empresa Enterprise", "av_outdated", "msp_cyber_security",
			"Servico de antivirus desatualizado em há 40 anos"),
		montarMock(agora.Add(-1*time.Hour), "Empresa Donklii (falencia)", "backup_warning", "cloudbackup",
			"Backup concluido com aviso (99999999 arquivos ignorados)"),
		montarMock(agora.Add(-2*time.Hour), "Empresa Do Malvado DOFENSMITH? (doofenshmirtz)", "high_latency", "msp_n_central",
			"Latencia anormal entre filiais Malvadas"),
	}
}

// ----- Internos -----

func montarMock(horario time.Time, cliente, tipo, produto, descricao string) Alerta {
	return Alerta{
		"client":          cliente,
		"type":            tipo,
		"product_keyword": produto,
		"created_at":      horario.Format(time.RFC3339),
		"message": map[string]any{
			"error":       descricao,
			"description": descricao,
		},
	}
}
