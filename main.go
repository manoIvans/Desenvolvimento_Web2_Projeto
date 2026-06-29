// main.go
//
// Entrypoint da API SignalHub. Mantido mínimo: embute o schema SQL (a
// diretiva //go:embed precisa morar no pacote main, na raiz) e delega todo
// o ciclo de vida ao pacote internal/aplicacao.

package main

import (
	"embed"

	"SignalHub/internal/aplicacao"
)

//go:embed db/schema
var esquemaSQL embed.FS

func main() {
	aplicacao.Executar(esquemaSQL)
}
