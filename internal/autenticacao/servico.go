// internal/autenticacao/servico.go
//
// Coordena a autenticação: emite um access token JWT (curto, validado de
// forma stateless no middleware) acompanhado de um refresh token opaco
// (longo, com estado em memória). O refresh token é de uso único — ao ser
// trocado em /refresh ele é invalidado e um novo par é emitido (rotação),
// limitando a janela de reuso de um token vazado. Mantém também um token
// mestre permanente configurado no boot. Comparações de segredo usam tempo
// constante para evitar timing attacks. Thread-safe.

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
	TTL_ACESSO_PADRAO   = time.Hour
	TTL_REFRESH_PADRAO  = 24 * time.Hour
	SUJEITO_PADRAO      = "operador"
)

// ----- Erros -----

// ErroNaoAutorizado é devolvido quando a senha de login não bate ou quando
// o refresh token é desconhecido/expirado. O handler HTTP o traduz para 401.
var ErroNaoAutorizado = errors.New("não autorizado")

// ----- Tipos -----

// Config agrupa os parâmetros de construção do Servico.
type Config struct {
	SegredoJWT  []byte
	TokenMestre string
	SenhaLogin  string
	TTLAcesso   time.Duration
	TTLRefresh  time.Duration
}

// Credenciais é o par de tokens emitido no login e na renovação.
type Credenciais struct {
	TokenAcesso  string
	TokenRefresh string
	ExpiraEm     time.Time
}

// sessaoRefresh guarda o dono e o vencimento de um refresh token ativo.
type sessaoRefresh struct {
	sujeito  string
	expiraEm time.Time
}

// Servico encapsula o segredo de assinatura JWT, o token mestre, a senha de
// login e o mapa de refresh tokens ativos em memória.
type Servico struct {
	mu          sync.RWMutex
	segredoJWT  []byte
	tokenMestre string
	senhaLogin  string
	ttlAcesso   time.Duration
	ttlRefresh  time.Duration
	refresh     map[string]sessaoRefresh
}

// NovoServico constrói o serviço a partir da configuração; TTLs ausentes ou
// não-positivos caem nos padrões.
func NovoServico(config Config) *Servico {
	if config.TTLAcesso <= 0 {
		config.TTLAcesso = TTL_ACESSO_PADRAO
	}
	if config.TTLRefresh <= 0 {
		config.TTLRefresh = TTL_REFRESH_PADRAO
	}
	return &Servico{
		segredoJWT:  config.SegredoJWT,
		tokenMestre: config.TokenMestre,
		senhaLogin:  config.SenhaLogin,
		ttlAcesso:   config.TTLAcesso,
		ttlRefresh:  config.TTLRefresh,
		refresh:     map[string]sessaoRefresh{},
	}
}

// ----- API pública -----

// Autenticar valida a senha em tempo constante e devolve um par de tokens.
func (s *Servico) Autenticar(senha string) (Credenciais, error) {
	if !comparaConstante(senha, s.senhaLogin) {
		return Credenciais{}, ErroNaoAutorizado
	}
	return s.emitirCredenciais(SUJEITO_PADRAO)
}

// Renovar troca um refresh token válido por um par novo. O refresh token
// recebido é invalidado no processo (rotação), mesmo que esteja expirado.
func (s *Servico) Renovar(tokenRefresh string) (Credenciais, error) {
	sessao, ok := s.consumirRefresh(tokenRefresh)
	if !ok {
		return Credenciais{}, ErroNaoAutorizado
	}
	return s.emitirCredenciais(sessao.sujeito)
}

// Revogar invalida um refresh token (logout). Não tem efeito sobre o token
// mestre nem sobre access tokens JWT já emitidos (stateless até expirarem).
func (s *Servico) Revogar(tokenRefresh string) {
	if tokenRefresh == "" {
		return
	}
	s.mu.Lock()
	delete(s.refresh, tokenRefresh)
	s.mu.Unlock()
}

// TokenValido devolve true se o token é o mestre ou um access token JWT com
// assinatura íntegra e ainda dentro da validade. Usado pelo middleware.
func (s *Servico) TokenValido(token string) bool {
	if token == "" {
		return false
	}
	if s.tokenMestre != "" && comparaConstante(token, s.tokenMestre) {
		return true
	}
	_, err := validarJWT(s.segredoJWT, token)
	return err == nil
}

// ----- Internos -----

// emitirCredenciais assina um access token JWT e registra um novo refresh
// token para o sujeito.
func (s *Servico) emitirCredenciais(sujeito string) (Credenciais, error) {
	tokenAcesso, expiraEm, err := gerarJWT(s.segredoJWT, sujeito, s.ttlAcesso)
	if err != nil {
		return Credenciais{}, err
	}

	tokenRefresh, err := gerarToken()
	if err != nil {
		return Credenciais{}, err
	}

	s.mu.Lock()
	s.refresh[tokenRefresh] = sessaoRefresh{
		sujeito:  sujeito,
		expiraEm: time.Now().UTC().Add(s.ttlRefresh),
	}
	s.mu.Unlock()

	return Credenciais{
		TokenAcesso:  tokenAcesso,
		TokenRefresh: tokenRefresh,
		ExpiraEm:     expiraEm,
	}, nil
}

// consumirRefresh remove o refresh token e devolve sua sessão. A remoção
// acontece mesmo quando ele já expirou — um refresh é sempre de uso único.
func (s *Servico) consumirRefresh(token string) (sessaoRefresh, bool) {
	if token == "" {
		return sessaoRefresh{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sessao, ok := s.refresh[token]
	if !ok {
		return sessaoRefresh{}, false
	}
	delete(s.refresh, token)

	if time.Now().UTC().After(sessao.expiraEm) {
		return sessaoRefresh{}, false
	}
	return sessao, true
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
