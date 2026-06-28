// internal/acronis/cliente.go
//
// Cliente HTTP para a API Acronis Cyber Protect Cloud (Cyber Platform API).
// Funções stateless que implementam o fluxo real documentado:
//   1. descoberta do datacenter   — GET https://cloud.acronis.com/api/1/accounts?login=...
//   2. token OAuth2 client_creds  — POST <server>/api/2/idp/token  (Basic auth)
//   3. listagem de alertas        — GET  <server>/api/alert_manager/v1/alerts (Bearer)
//
// O ciclo de vida do token (cache + reemissão por expires_on) vive no
// Servico — aqui só há chamadas puras.

package acronis

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ----- Constantes -----

const (
	TIMEOUT_HTTP = 15 * time.Second

	URL_DESCOBERTA_PADRAO = "https://cloud.acronis.com"
	ROTA_DESCOBERTA       = "/api/1/accounts"
	ROTA_TOKEN            = "/api/2/idp/token"
	ROTA_ALERTAS          = "/api/alert_manager/v1/alerts"

	// LIMITE_ALERTAS fixa o teto de alertas pedidos numa chamada (parâmetro
	// `limit`), em vez de depender do default do servidor. A API não oferece
	// cursor de continuação, então este é o controle de volume disponível.
	LIMITE_ALERTAS = 1000

	GRANT_CLIENT_CREDENTIALS = "client_credentials"
	PREFIXO_BEARER           = "Bearer "
	TIPO_TOKEN_BEARER        = "bearer"
	CONTENT_TYPE_FORM        = "application/x-www-form-urlencoded"
)

// ----- Erros -----

// erroTokenExpirado sinaliza um 401 em chamada autenticada — o Servico o usa
// como gatilho para reemitir o token e repetir a requisição.
var erroTokenExpirado = errors.New("token Acronis expirado ou inválido (401)")

// ----- Cliente compartilhado -----

// httpClientPadrao é compartilhado entre chamadas. Diferente de Zabbix/MSP,
// a Acronis é nuvem pública com certificados válidos — não se desativa a
// verificação TLS.
var httpClientPadrao = &http.Client{Timeout: TIMEOUT_HTTP}

// ----- API interna -----

// descobrirServidor resolve a URL base do datacenter de um login via endpoint
// fixo de descoberta. A Acronis recomenda nunca fixar a URL do datacenter.
func descobrirServidor(cli *http.Client, urlDescoberta, login string) (string, error) {
	endereco := fmt.Sprintf("%s%s?login=%s", urlDescoberta, ROTA_DESCOBERTA, url.QueryEscape(login))

	req, err := http.NewRequest(http.MethodGet, endereco, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("descoberta de datacenter Acronis: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("descoberta Acronis devolveu status %d", resp.StatusCode)
	}

	var corpo respostaDescoberta
	if err := json.NewDecoder(resp.Body).Decode(&corpo); err != nil {
		return "", fmt.Errorf("resposta de descoberta inválida: %w", err)
	}
	if corpo.ServerURL == "" {
		return "", fmt.Errorf("descoberta Acronis não retornou server_url")
	}
	return strings.TrimRight(corpo.ServerURL, "/"), nil
}

// obterToken executa o fluxo OAuth2 client_credentials e devolve o token de
// acesso com seu instante de expiração (expires_on).
func obterToken(cli *http.Client, serverURL, clientID, clientSecret string) (respostaToken, error) {
	formulario := url.Values{"grant_type": {GRANT_CLIENT_CREDENTIALS}}

	req, err := http.NewRequest(http.MethodPost, serverURL+ROTA_TOKEN, strings.NewReader(formulario.Encode()))
	if err != nil {
		return respostaToken{}, err
	}
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", CONTENT_TYPE_FORM)
	req.Header.Set("Accept", "application/json")

	resp, err := cli.Do(req)
	if err != nil {
		return respostaToken{}, fmt.Errorf("requisição de token Acronis: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return respostaToken{}, fmt.Errorf("credenciais Acronis inválidas ou expiradas (401)")
	}
	if resp.StatusCode != http.StatusOK {
		return respostaToken{}, fmt.Errorf("token Acronis devolveu status %d", resp.StatusCode)
	}

	var token respostaToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return respostaToken{}, fmt.Errorf("resposta de token inválida: %w", err)
	}
	if token.AccessToken == "" {
		return respostaToken{}, fmt.Errorf("token Acronis veio vazio")
	}
	if token.TokenType != "" && !strings.EqualFold(token.TokenType, TIPO_TOKEN_BEARER) {
		return respostaToken{}, fmt.Errorf("token Acronis com token_type inesperado: %q", token.TokenType)
	}
	return token, nil
}

// buscarAlertas chama GET /api/alert_manager/v1/alerts autenticado por Bearer
// e devolve os alertas brutos. Um 401 vira erroTokenExpirado.
func buscarAlertas(cli *http.Client, serverURL, token string) ([]alertaRaw, error) {
	parametros := url.Values{"limit": {strconv.Itoa(LIMITE_ALERTAS)}}

	req, err := http.NewRequest(http.MethodGet, serverURL+ROTA_ALERTAS+"?"+parametros.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", PREFIXO_BEARER+token)
	req.Header.Set("Accept", "application/json")

	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requisição de alertas Acronis: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, erroTokenExpirado
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alertas Acronis devolveram status %d", resp.StatusCode)
	}

	corpo, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var envelope respostaAlertas
	if err := json.Unmarshal(corpo, &envelope); err != nil {
		return nil, fmt.Errorf("resposta de alertas inválida: %w", err)
	}
	return envelope.Items, nil
}
