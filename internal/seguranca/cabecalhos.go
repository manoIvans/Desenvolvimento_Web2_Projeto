// internal/seguranca/cabecalhos.go
//
// Middleware que adiciona cabeçalhos de segurança a todas as respostas
// (correção OWASP — A05:2021 Security Misconfiguration). Endurece o
// navegador contra sniffing de MIME, clickjacking, vazamento de referrer e
// injeção de conteúdo (CSP).

package seguranca

import "net/http"

// ----- Constantes -----

const (
	// Todo o frontend usa scripts e estilos externos (sem inline), então a
	// CSP restringe scripts e estilos à própria origem — sem 'unsafe-inline'.
	// object-src/base-uri/frame-ancestors fecham vetores comuns.
	CSP_PADRAO = "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self'; " +
		"img-src 'self' data:; " +
		"object-src 'none'; " +
		"base-uri 'self'; " +
		"frame-ancestors 'none'"

	// HSTS só tem efeito sobre HTTPS; em HTTP o navegador ignora — manter o
	// header é inofensivo e correto para quando houver TLS na frente.
	HSTS_PADRAO = "max-age=31536000; includeSubDomains"
)

// ----- API pública -----

// CabecalhosSeguranca devolve um middleware que injeta os cabeçalhos de
// segurança antes de delegar ao próximo handler.
func CabecalhosSeguranca(proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cabecalho := w.Header()
		cabecalho.Set("X-Content-Type-Options", "nosniff")
		cabecalho.Set("X-Frame-Options", "DENY")
		cabecalho.Set("Referrer-Policy", "no-referrer")
		cabecalho.Set("Content-Security-Policy", CSP_PADRAO)
		cabecalho.Set("Strict-Transport-Security", HSTS_PADRAO)
		proximo.ServeHTTP(w, r)
	})
}
