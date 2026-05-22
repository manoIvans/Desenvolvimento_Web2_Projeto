// internal/autenticacao/servico.go
//
// Gerencia o token mestre (permanente, configurado no boot) e as sessões
// temporárias criadas por POST /login. Comparações de segredo usam tempo
// constante para evitar timing attacks; o cleanup de sessões expiradas é
// preguiçoso, executado dentro de TokenValido.

package autenticacao

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// ----- Constantes -----

const (
	TAMANHO_TOKEN_BYTES = 32
	TTL_PADRAO          = time.Hour
)

// ----- Erros -----

// ErroNaoAutorizado é devolvido por Autenticar quando a senha não bate.
// O handler HTTP o traduz para 401.
var ErroNaoAutorizado = errors.New("não autorizado")

// ----- Tipo Servico -----

// Servico encapsula o token mestre, a senha de login e o mapa de sessões
// temporárias em memória. Thread-safe.
type Servico struct {
	mu          sync.RWMutex
	tokenMestre string
	senhaLogin  string
	ttl         time.Duration
	sessoes     map[string]time.Time
}

// NovoServico constrói o serviço com o token mestre, a senha aceita em
// POST /login e o TTL aplicado a cada sessão emitida.
func NovoServico(tokenMestre, senhaLogin string, ttl time.Duration) *Servico {
	if ttl <= 0 {
		ttl = TTL_PADRAO
	}
	return &Servico{
		tokenMestre: tokenMestre,
		senhaLogin:  senhaLogin,
		ttl:         ttl,
		sessoes:     map[string]time.Time{},
	}
}

// ----- API pública -----

// Autenticar valida a senha em tempo constante e devolve um token
// recém-emitido + o instante de expiração.
func (s *Servico) Autenticar(senha string) (string, time.Time, error) {
	if !comparaConstante(senha, s.senhaLogin) {
		return "", time.Time{}, ErroNaoAutorizado
	}

	token, err := gerarToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiraEm := time.Now().UTC().Add(s.ttl)

	s.mu.Lock()
	s.sessoes[token] = expiraEm
	s.mu.Unlock()

	return token, expiraEm, nil
}

// TokenValido devolve true se o token bate com o mestre ou com uma sessão
// temporária ainda válida. Sessões expiradas são removidas no caminho.
func (s *Servico) TokenValido(token string) bool {
	if token == "" {
		return false
	}
	if comparaConstante(token, s.tokenMestre) {
		return true
	}

	s.mu.RLock()
	expiraEm, ok := s.sessoes[token]
	s.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().UTC().After(expiraEm) {
		s.descartar(token)
		return false
	}
	return true
}

// Revogar remove uma sessão temporária. Não tem efeito sobre o token mestre.
func (s *Servico) Revogar(token string) {
	if token == "" || comparaConstante(token, s.tokenMestre) {
		return
	}
	s.descartar(token)
}

// ----- Internos -----

func (s *Servico) descartar(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessoes, token)
}

// ----- Utilitários -----

func gerarToken() (string, error) {
	buf := make([]byte, TAMANHO_TOKEN_BYTES)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func comparaConstante(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
