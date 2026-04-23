// internal/zabbix/tipos.go
//
// Tipos de saída (frontend) e tipos internos da API JSON-RPC Zabbix.

package zabbix

import "encoding/json"

// ----- Tipos de saída -----

// ProblemaTag é uma tag aplicada a um problema/trigger.
type ProblemaTag struct {
	Tag   string `json:"tag"`
	Value string `json:"value,omitempty"`
}

// Problema é o formato consumido pelo frontend — um problema consolidado de
// qualquer instância Zabbix configurada.
type Problema struct {
	Grupo     string        `json:"grupo"`
	Host      string        `json:"host"`
	Hosts     []string      `json:"hosts,omitempty"`
	Grupos    []string      `json:"grupos,omitempty"`
	Evento    string        `json:"evento"`
	Mensagem  string        `json:"mensagem"`
	PrioLabel string        `json:"prio_label,omitempty"`
	Horario   string        `json:"horario"`
	Source    string        `json:"source,omitempty"`
	Instancia string        `json:"instancia,omitempty"`
	URL       string        `json:"url,omitempty"`
	Tags      []ProblemaTag `json:"tags,omitempty"`
}

// ResultadoRefresh é o que o endpoint /zabbix/refresh devolve: dados + falhas
// por instância + versões reportadas.
type ResultadoRefresh struct {
	Data    []Problema        `json:"data"`
	Falhas  []string          `json:"falhas,omitempty"`
	Versoes map[string]string `json:"versoes,omitempty"`
}

// ----- Tipos internos da API JSON-RPC -----

type apiReq struct {
	Jsonrpc string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
	Auth    string         `json:"auth,omitempty"`
	ID      int            `json:"id"`
}

type apiResp struct {
	Result json.RawMessage `json:"result"`
	Error  *apiError       `json:"error"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type triggerRaw struct {
	TriggerID   string      `json:"triggerid"`
	Description string      `json:"description"`
	Priority    json.Number `json:"priority"`
	LastChange  json.Number `json:"lastchange"`
	Comments    string      `json:"comments"`
	URL         string      `json:"url"`
	OpData      string      `json:"opdata"`
	Hosts       []struct {
		HostID string `json:"hostid"`
		Host   string `json:"host"`
		Name   string `json:"name"`
	} `json:"hosts"`
	Groups []struct {
		GroupID string `json:"groupid"`
		Name    string `json:"name"`
	} `json:"groups"`
	Tags []struct {
		Tag   string `json:"tag"`
		Value string `json:"value"`
	} `json:"tags"`
}
