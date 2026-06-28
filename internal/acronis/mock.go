// internal/acronis/mock.go
//
// Dados de demonstração usados quando não há conta Acronis configurada.
// O entrypoint injeta esses alertas no cache via Servico.AtivarModoDemo,
// e o GET /acronis passa a retorná-los como se viessem da API real. A forma
// de cada mock espelha o que converterAlerta produziria a partir de um
// alerta bruto do Alert Manager (severity, type CTI, details, tenant).

package acronis

import "time"

// ----- Constantes -----

const (
	TIPO_CTI_BACKUP  = "cti.a.p.am.alert.v1.0~a.p.basic.v1.0~vendor.application.backup_failed.v1.0"
	TIPO_CTI_MALWARE = "cti.a.p.am.alert.v1.0~a.p.basic.v1.0~vendor.application.malware_detected.v1.0"
	TIPO_CTI_AGENTE  = "cti.a.p.am.alert.v1.0~a.p.basic.v1.0~vendor.application.agent_offline.v1.0"
)

// ----- API pública -----

// MocksDemonstracao devolve uma lista fixa de alertas para o modo demo.
// Os horários são calculados em relação a `agora` para parecerem recentes.
func MocksDemonstracao() []Alerta {
	agora := time.Now().UTC()
	return []Alerta{
		montarMock(agora.Add(-6*time.Minute), "a1b2c3d4-0001", TIPO_CTI_MALWARE, "critical", "Proteção",
			"Malware detectado", "Arquivo malicioso \"trojan.exe\" colocado em quarentena", "Empresa Farma Aura",
			map[string]any{"Tipo de malware": "Trojan:Win32/Caphaw", "Dispositivo": "PC-RECEPCAO"}),
		montarMock(agora.Add(-18*time.Minute), "a1b2c3d4-0002", TIPO_CTI_BACKUP, "error", "Backup",
			"Falha no backup", "Backup diário do servidor de arquivos falhou (espaço insuficiente)", "Empresa Betinha corp",
			map[string]any{"Plano": "Backup Diário", "Recurso": "SRV-ARQUIVOS"}),
		montarMock(agora.Add(-40*time.Minute), "a1b2c3d4-0003", TIPO_CTI_AGENTE, "warning", "Dispositivos",
			"Agente offline", "O agente de proteção está offline há mais de 30 minutos", "Empresa Enterprise",
			map[string]any{"Dispositivo": "NOTE-DIRETORIA", "Última conexão": "há 31 min"}),
		montarMock(agora.Add(-1*time.Hour-10*time.Minute), "a1b2c3d4-0004", TIPO_CTI_BACKUP, "information", "Backup",
			"Backup concluído com aviso", "Backup concluído, porém 12 arquivos foram ignorados", "Empresa Donklii (falência)",
			map[string]any{"Plano": "Backup Semanal", "Ignorados": "12"}),
	}
}

// ----- Internos -----

func montarMock(horario time.Time, id, tipo, severidade, categoria, titulo, descricao, tenant string, detalhes map[string]any) Alerta {
	return Alerta{
		ID:         id,
		Tipo:       tipo,
		Categoria:  categoria,
		Severidade: severidade,
		Titulo:     titulo,
		Descricao:  descricao,
		Detalhes:   detalhes,
		Horario:    horario.Format(time.RFC3339),
		Tenant:     tenant,
		Source:     SOURCE_API_ACRONIS,
	}
}
