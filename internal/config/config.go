// internal/config/config.go
//
// Leitura do configuracoes.json (instâncias Zabbix e MSP Clouds).
// Sprint 1: só JSON em disco. Sprint 2 moverá as instâncias para Postgres.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const NOME_ARQUIVO_CONFIG = "configuracoes.json"

// ----- Tipos -----

// InstanciaZabbix representa uma instância Zabbix configurada.
type InstanciaZabbix struct {
	Nome   string `json:"nome"`
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// Config é o conteúdo do configuracoes.json.
type Config struct {
	ZabbixInstancias []InstanciaZabbix `json:"zabbix_instancias"`
	MspInstancias    []string          `json:"msp_instancias"`
	PortaWeb         string            `json:"porta_web,omitempty"`
}

// ----- Leitura -----

// Ler tenta carregar o configuracoes.json do diretório do executável
// (ou CWD quando rodando via "go run"). Retorna Config vazia se o arquivo
// não existir — os handlers devem tolerar listas vazias.
func Ler() (Config, error) {
	caminho, err := resolverCaminho()
	if err != nil {
		return Config{}, err
	}

	dados, err := os.ReadFile(caminho)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("ler %s: %w", caminho, err)
	}

	var cfg Config
	if err := json.Unmarshal(dados, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", caminho, err)
	}
	return cfg, nil
}

// ----- Resolução de caminho -----

// resolverCaminho devolve o caminho absoluto do configuracoes.json —
// ao lado do executável em produção, ou no CWD quando rodando via "go run".
func resolverCaminho() (string, error) {
	executavel, err := os.Executable()
	if err != nil {
		return "", err
	}
	diretorioBase := filepath.Dir(executavel)

	if ehGoRunTemp(diretorioBase) {
		diretorioBase, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(diretorioBase, NOME_ARQUIVO_CONFIG), nil
}

// ehGoRunTemp detecta TEMP do sistema ou GOCACHE (go-build/...).
func ehGoRunTemp(dir string) bool {
	if tmp := os.TempDir(); tmp != "" && strings.HasPrefix(dir, tmp) {
		return true
	}
	return strings.Contains(dir, string(filepath.Separator)+"go-build")
}
