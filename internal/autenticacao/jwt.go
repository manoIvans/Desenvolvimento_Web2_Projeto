// internal/autenticacao/jwt.go
//
// Implementação mínima de JWT assinado com HS256 (HMAC-SHA256) usando só a
// biblioteca padrão — sem dependências externas. Gera o access token do
// login/refresh e o valida no middleware de forma stateless (assinatura +
// expiração). O refresh token continua opaco e com estado (ver servico.go).

package autenticacao

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ----- Constantes -----

const (
	ALGORITMO_JWT      = "HS256"
	TIPO_CABECALHO_JWT = "JWT"
	TIPO_TOKEN_ACESSO  = "acesso"
	SEGMENTOS_JWT      = 3
)

// ----- Erros -----

// ErroJWTInvalido cobre qualquer falha de validação do token: formato,
// algoritmo inesperado, assinatura adulterada ou expiração.
var ErroJWTInvalido = errors.New("jwt inválido")

// ----- Tipos -----

// cabecalhoJWT é o header JOSE serializado no primeiro segmento.
type cabecalhoJWT struct {
	Algoritmo string `json:"alg"`
	Tipo      string `json:"typ"`
}

// ReivindicacoesJWT são as claims do payload (segundo segmento).
type ReivindicacoesJWT struct {
	Sujeito   string `json:"sub"`
	Tipo      string `json:"tipo"`
	ID        string `json:"jti"`
	EmitidoEm int64  `json:"iat"`
	ExpiraEm  int64  `json:"exp"`
}

// ----- API interna do pacote -----

// gerarJWT assina um access token para o sujeito informado, com validade
// igual a ttl. Devolve o token compacto e o instante exato de expiração.
func gerarJWT(segredo []byte, sujeito string, ttl time.Duration) (string, time.Time, error) {
	agora := time.Now().UTC()
	expiraEm := agora.Add(ttl)

	identificador, err := gerarToken()
	if err != nil {
		return "", time.Time{}, err
	}

	cabecalhoCodificado, err := codificarSegmento(cabecalhoJWT{
		Algoritmo: ALGORITMO_JWT,
		Tipo:      TIPO_CABECALHO_JWT,
	})
	if err != nil {
		return "", time.Time{}, err
	}

	reivindicacoesCodificadas, err := codificarSegmento(ReivindicacoesJWT{
		Sujeito:   sujeito,
		Tipo:      TIPO_TOKEN_ACESSO,
		ID:        identificador,
		EmitidoEm: agora.Unix(),
		ExpiraEm:  expiraEm.Unix(),
	})
	if err != nil {
		return "", time.Time{}, err
	}

	conteudo := cabecalhoCodificado + "." + reivindicacoesCodificadas
	return conteudo + "." + assinar(segredo, conteudo), expiraEm, nil
}

// validarJWT confere a assinatura em tempo constante, o algoritmo e a
// expiração. Devolve as claims quando o token é válido.
func validarJWT(segredo []byte, token string) (ReivindicacoesJWT, error) {
	partes := strings.Split(token, ".")
	if len(partes) != SEGMENTOS_JWT {
		return ReivindicacoesJWT{}, ErroJWTInvalido
	}

	conteudo := partes[0] + "." + partes[1]
	if !assinaturaConfere(segredo, conteudo, partes[2]) {
		return ReivindicacoesJWT{}, ErroJWTInvalido
	}

	var cabecalho cabecalhoJWT
	if err := decodificarSegmento(partes[0], &cabecalho); err != nil {
		return ReivindicacoesJWT{}, ErroJWTInvalido
	}
	if cabecalho.Algoritmo != ALGORITMO_JWT {
		return ReivindicacoesJWT{}, ErroJWTInvalido
	}

	var reivindicacoes ReivindicacoesJWT
	if err := decodificarSegmento(partes[1], &reivindicacoes); err != nil {
		return ReivindicacoesJWT{}, ErroJWTInvalido
	}
	if time.Now().UTC().Unix() >= reivindicacoes.ExpiraEm {
		return ReivindicacoesJWT{}, ErroJWTInvalido
	}
	return reivindicacoes, nil
}

// ----- Utilitários -----

func assinar(segredo []byte, conteudo string) string {
	mac := hmac.New(sha256.New, segredo)
	mac.Write([]byte(conteudo))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func assinaturaConfere(segredo []byte, conteudo, assinaturaRecebida string) bool {
	esperada := assinar(segredo, conteudo)
	return subtle.ConstantTimeCompare([]byte(esperada), []byte(assinaturaRecebida)) == 1
}

func codificarSegmento(valor any) (string, error) {
	bruto, err := json.Marshal(valor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bruto), nil
}

func decodificarSegmento(segmento string, destino any) error {
	bruto, err := base64.RawURLEncoding.DecodeString(segmento)
	if err != nil {
		return err
	}
	return json.Unmarshal(bruto, destino)
}
