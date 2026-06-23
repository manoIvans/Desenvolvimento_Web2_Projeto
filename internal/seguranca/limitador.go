// internal/seguranca/limitador.go
//
// Rate limiting por IP de origem com algoritmo token bucket (correção OWASP
// — protege /login e /refresh contra força bruta e enumeração de senha).
// Cada IP tem um balde que recarrega ao longo do tempo; quando esvazia, as
// requisições recebem 429. Thread-safe.

package seguranca

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"SignalHub/internal/resposta"
)

// ----- Tipos -----

// balde guarda os tokens disponíveis de um IP e o instante da última recarga.
type balde struct {
	tokens        float64
	ultimaRecarga time.Time
}

// Limitador aplica token bucket por chave (IP). capacidade é o pico de
// rajada; recargaPorSegundo é a taxa sustentada de reposição de tokens.
type Limitador struct {
	mu                sync.Mutex
	baldes            map[string]*balde
	capacidade        float64
	recargaPorSegundo float64
	relogio           func() time.Time
}

// ----- Construtor -----

// NovoLimitador cria um limitador com a capacidade (rajada) e a taxa de
// recarga por segundo informadas.
func NovoLimitador(capacidade int, recargaPorSegundo float64) *Limitador {
	return &Limitador{
		baldes:            map[string]*balde{},
		capacidade:        float64(capacidade),
		recargaPorSegundo: recargaPorSegundo,
		relogio:           time.Now,
	}
}

// ----- API pública -----

// Middleware aplica o rate limiting por IP de origem. Excedido o limite,
// responde 429 com Retry-After.
func (l *Limitador) Middleware(proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Permitir(ipDeOrigem(r)) {
			w.Header().Set("Retry-After", strconv.Itoa(l.segundosPorToken()))
			resposta.Erro(w, http.StatusTooManyRequests, "muitas requisições — tente novamente em instantes")
			return
		}
		proximo.ServeHTTP(w, r)
	})
}

// Permitir tenta consumir um token da chave. Devolve false quando o balde
// está vazio (limite estourado).
func (l *Limitador) Permitir(chave string) bool {
	agora := l.relogio()

	l.mu.Lock()
	defer l.mu.Unlock()

	atual, existe := l.baldes[chave]
	if !existe {
		l.baldes[chave] = &balde{tokens: l.capacidade - 1, ultimaRecarga: agora}
		return true
	}

	decorrido := agora.Sub(atual.ultimaRecarga).Seconds()
	atual.tokens = min(l.capacidade, atual.tokens+decorrido*l.recargaPorSegundo)
	atual.ultimaRecarga = agora

	if atual.tokens < 1 {
		return false
	}
	atual.tokens--
	return true
}

// ----- Utilitários -----

func (l *Limitador) segundosPorToken() int {
	if l.recargaPorSegundo <= 0 {
		return 1
	}
	return int(1 / l.recargaPorSegundo)
}

func ipDeOrigem(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
