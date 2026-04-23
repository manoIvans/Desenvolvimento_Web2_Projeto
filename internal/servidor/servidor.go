// internal/servidor/servidor.go
//
// Composição do roteador (middlewares globais + rotas dos domínios) e
// inicialização/parada elegante do HTTP server.

package servidor

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"SignalHub/internal/mspclouds"
	"SignalHub/internal/saude"
	"SignalHub/internal/zabbix"
)

const TIMEOUT_SHUTDOWN = 15 * time.Second

// Dependencias agrupa os handlers injetados no servidor.
type Dependencias struct {
	HandlerZabbix *zabbix.Handler
	HandlerMsp    *mspclouds.Handler
}

// ----- Router -----

// MontarRouter constrói o *chi.Mux com middlewares e rotas de todos os domínios.
// Exportado para permitir testes via httptest sem subir servidor TCP.
func MontarRouter(deps Dependencias) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	saude.Rotas(r)
	deps.HandlerZabbix.Rotas(r)
	deps.HandlerMsp.Rotas(r)

	return r
}

// ----- Servidor -----

// Servidor encapsula o *http.Server com métodos Iniciar/Parar thread-safe.
type Servidor struct {
	mu         sync.Mutex
	httpServer *http.Server
	endereco   string
	handler    http.Handler
}

// Novo cria um Servidor configurado mas ainda não iniciado.
func Novo(endereco string, handler http.Handler) *Servidor {
	return &Servidor{
		endereco: endereco,
		handler:  handler,
	}
}

// Iniciar sobe o HTTP server e bloqueia até Parar ou erro fatal.
// Retorna nil em shutdown limpo.
func (s *Servidor) Iniciar() error {
	s.mu.Lock()
	s.httpServer = &http.Server{
		Addr:              s.endereco,
		Handler:           s.handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	s.mu.Unlock()

	slog.Info("servidor http escutando", "endereco", s.endereco)
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Parar executa shutdown elegante com timeout interno.
func (s *Servidor) Parar(contexto context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.httpServer == nil {
		return nil
	}
	ctxShutdown, cancelar := context.WithTimeout(contexto, TIMEOUT_SHUTDOWN)
	defer cancelar()
	return s.httpServer.Shutdown(ctxShutdown)
}
