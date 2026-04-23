// internal/mspclouds/cliente.go
//
// Cliente HTTP para a API MSP Clouds. Um "alerta" é um mapa genérico
// (map[string]any) porque o formato upstream tem muitos campos dinâmicos
// por produto — o frontend interpreta conforme o "type" e "product_keyword".

package mspclouds

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	URL_BASE_PADRAO = "https://mspclouds.com"
	TIMEOUT_HTTP    = 15 * time.Second
)

// httpClientPadrao é compartilhado entre chamadas.
var httpClientPadrao = &http.Client{
	Timeout: TIMEOUT_HTTP,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	},
}

// Alerta é um alerta retornado pela API MSP Clouds no formato bruto.
// Usamos map[string]any para passar-through os campos ao frontend.
type Alerta = map[string]any

// buscarAlertas chama GET /api/v1/alerts?api_key=... e devolve a lista.
func buscarAlertas(cli *http.Client, baseURL, apiKey string) ([]Alerta, error) {
	url := fmt.Sprintf("%s/api/v1/alerts?api_key=%s", baseURL, apiKey)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requisição MSP alertas: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MSP alertas devolveu status %d", resp.StatusCode)
	}

	corpo, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lista []Alerta
	if err := json.Unmarshal(corpo, &lista); err != nil {
		return nil, fmt.Errorf("resposta MSP inválida: %w", err)
	}
	return lista, nil
}
