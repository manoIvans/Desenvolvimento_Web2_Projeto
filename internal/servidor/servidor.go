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

	"SignalHub/internal/frontend"
	"SignalHub/internal/mspclouds"
	"SignalHub/internal/saude"
	"SignalHub/internal/zabbix"
)

// ----- Constantes -----

const (
	TIMEOUT_SHUTDOWN    = 15 * time.Second
	TIMEOUT_READ_HEADER = 10 * time.Second
	CORS_MAX_AGE        = 300
)

var CORS_METODOS_PERMITIDOS = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
var CORS_HEADERS_PERMITIDOS = []string{"Content-Type", "Authorization"}
var CORS_ORIGENS_PERMITIDAS = []string{"*"}

// ----- Tipos -----

// Dependencias agrupa os handlers injetados no servidor.
type Dependencias struct {
	HandlerZabbix *zabbix.Handler
	HandlerMsp    *mspclouds.Handler
}

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

// ----- Router -----

// MontarRouter constrói o *chi.Mux com middlewares e rotas de todos os domínios.
// Exportado para permitir testes via httptest sem subir servidor TCP.
func MontarRouter(deps Dependencias) http.Handler {
	r := chi.NewRouter()
	registrarMiddlewares(r)
	registrarRotas(r, deps)
	return r
}

func registrarMiddlewares(r chi.Router) {
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   CORS_ORIGENS_PERMITIDAS,
		AllowedMethods:   CORS_METODOS_PERMITIDOS,
		AllowedHeaders:   CORS_HEADERS_PERMITIDOS,
		AllowCredentials: false,
		MaxAge:           CORS_MAX_AGE,
	}))
}

func registrarRotas(r chi.Router, deps Dependencias) {
	saude.Rotas(r)
	deps.HandlerZabbix.Rotas(r)
	deps.HandlerMsp.Rotas(r)
	frontend.Rotas(r)
}

// ----- Ciclo de vida -----

// Iniciar sobe o HTTP server e bloqueia até Parar ou erro fatal.
// Retorna nil em shutdown limpo.
func (s *Servidor) Iniciar() error {
	s.mu.Lock()
	s.httpServer = &http.Server{
		Addr:              s.endereco,
		Handler:           s.handler,
		ReadHeaderTimeout: TIMEOUT_READ_HEADER,
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
