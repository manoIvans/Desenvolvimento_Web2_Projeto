// internal/frontend/http.go
//
// Serve os arquivos estáticos da pasta `frontend/` no endpoint raiz `/`.
// O caminho do diretório é resolvido relativo ao executável (produção) ou
// ao CWD (durante `go run`).

package frontend

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// ----- Constantes -----

const (
	NOME_PASTA_FRONTEND    = "frontend"
	PASTA_GO_BUILD         = "go-build"
	ROTA_INDEX             = "/"
	ROTA_ESTATICOS         = "/*"
	VAR_AMBIENTE_DIR_FRONT = "SIGNALHUB_FRONTEND"
)

// ----- API pública -----

// Rotas registra o file server estático em `/*`. Se o diretório não for
// localizado, loga warning e segue — a API JSON continua funcional.
func Rotas(r chi.Router) {
	caminho := resolverDiretorioFrontend()
	if caminho == "" {
		slog.Warn("frontend não encontrado — endpoint / não será servido")
		return
	}
	slog.Info("frontend servido", "diretorio", caminho)

	servidorArquivos := http.FileServer(http.Dir(caminho))
	r.Get(ROTA_INDEX, servidorArquivos.ServeHTTP)
	r.Get(ROTA_ESTATICOS, servidorArquivos.ServeHTTP)
}

// ----- Internos -----

// resolverDiretorioFrontend devolve o caminho absoluto da pasta `frontend/`
// procurando, em ordem:
//  1. Override via env var SIGNALHUB_FRONTEND
//  2. Ao lado do executável         (<bin>/frontend)
//  3. Um nível acima do executável  (<bin>/../frontend) — útil para `bin/signalhub`
//  4. No CWD                        (<cwd>/frontend)   — útil em `go run`
func resolverDiretorioFrontend() string {
	if override := os.Getenv(VAR_AMBIENTE_DIR_FRONT); override != "" && dirExiste(override) {
		return override
	}

	for _, candidato := range candidatosFrontend() {
		if dirExiste(candidato) {
			return candidato
		}
	}
	return ""
}

func candidatosFrontend() []string {
	var lista []string

	executavel, err := os.Executable()
	if err == nil {
		base := filepath.Dir(executavel)
		if ehGoRunTemp(base) {
			if cwd, errCwd := os.Getwd(); errCwd == nil {
				base = cwd
			}
		}
		lista = append(lista,
			filepath.Join(base, NOME_PASTA_FRONTEND),
			filepath.Join(filepath.Dir(base), NOME_PASTA_FRONTEND),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		lista = append(lista, filepath.Join(cwd, NOME_PASTA_FRONTEND))
	}
	return lista
}

// ehGoRunTemp detecta TEMP do sistema ou GOCACHE (go-build/...).
func ehGoRunTemp(dir string) bool {
	if tmp := os.TempDir(); tmp != "" && strings.HasPrefix(dir, tmp) {
		return true
	}
	return strings.Contains(dir, string(filepath.Separator)+PASTA_GO_BUILD)
}

func dirExiste(caminho string) bool {
	info, err := os.Stat(caminho)
	if err != nil {
		return false
	}
	return info.IsDir()
}
