// internal/zabbix/cliente.go
//
// Cliente HTTP para a API JSON-RPC Zabbix. Tenta primeiro o método moderno
// (Authorization: Bearer, Zabbix 6.4+) e cai para o legado (auth no body)
// em caso de falha.

package zabbix

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ----- Constantes -----

const (
	TIMEOUT_HTTP       = 15 * time.Second
	JSONRPC_VERSAO     = "2.0"
	CONTENT_TYPE_JSON  = "application/json"
	HEADER_AUTORIZACAO = "Authorization"
	PREFIXO_BEARER     = "Bearer "
)

// ----- Cliente compartilhado -----

// httpClientPadrao é o cliente compartilhado por todas as chamadas.
// InsecureSkipVerify: true porque muitas instâncias Zabbix internas usam
// certificados auto-assinados — risco aceitável para esse contexto.
var httpClientPadrao = &http.Client{
	Timeout: TIMEOUT_HTTP,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	},
}

// ----- API interna -----

// chamarApiAdaptado tenta o método moderno primeiro e cai para o legado se falhar.
func chamarApiAdaptado(cli *http.Client, apiURL, apiKey, metodo string, params map[string]any, destino any) error {
	if err := chamarApiModerno(cli, apiURL, apiKey, metodo, params, destino); err == nil {
		return nil
	}
	return chamarApiLegado(cli, apiURL, apiKey, metodo, params, destino)
}

// chamarApiModerno envia auth via header Authorization: Bearer (Zabbix 6.4+).
func chamarApiModerno(cli *http.Client, apiURL, apiKey, metodo string, params map[string]any, destino any) error {
	corpo, err := json.Marshal(apiReq{
		Jsonrpc: JSONRPC_VERSAO,
		Method:  metodo,
		Params:  params,
		ID:      1,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_JSON)
	req.Header.Set(HEADER_AUTORIZACAO, PREFIXO_BEARER+apiKey)

	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("requisição %s: %w", metodo, err)
	}
	defer resp.Body.Close()

	return decodificarResposta(resp.Body, metodo, destino)
}

// chamarApiLegado envia auth no body (Zabbix < 6.4).
func chamarApiLegado(cli *http.Client, apiURL, apiKey, metodo string, params map[string]any, destino any) error {
	corpo, err := json.Marshal(apiReq{
		Jsonrpc: JSONRPC_VERSAO,
		Method:  metodo,
		Params:  params,
		Auth:    apiKey,
		ID:      1,
	})
	if err != nil {
		return err
	}

	resp, err := cli.Post(apiURL, CONTENT_TYPE_JSON, bytes.NewReader(corpo))
	if err != nil {
		return fmt.Errorf("requisição %s: %w", metodo, err)
	}
	defer resp.Body.Close()

	return decodificarResposta(resp.Body, metodo, destino)
}

// ----- Helpers -----

// decodificarResposta faz o parse do envelope JSON-RPC e retorna o resultado
// (ou erro da API se Error estiver presente).
func decodificarResposta(body io.Reader, metodo string, destino any) error {
	dados, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	var envelope apiResp
	if err := json.Unmarshal(dados, &envelope); err != nil {
		return fmt.Errorf("resposta inválida de %s: %w", metodo, err)
	}
	if envelope.Error != nil {
		return fmt.Errorf("API erro %d em %s: %s — %s",
			envelope.Error.Code, metodo, envelope.Error.Message, envelope.Error.Data)
	}
	return json.Unmarshal(envelope.Result, destino)
}
