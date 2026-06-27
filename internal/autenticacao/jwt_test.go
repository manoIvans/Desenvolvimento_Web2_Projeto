// internal/autenticacao/jwt_test.go
//
// Testes do gerador/validador de JWT HS256: roundtrip, rejeição de
// assinatura adulterada, segredo errado, expiração, formato inválido e
// confusão de algoritmo (alg != HS256).

package autenticacao

import (
	"testing"
	"time"
)

// ----- Constantes de teste -----

var segredoJWTDemo = []byte("segredo-jwt-de-teste")

// ----- Testes -----

func TestGerarEValidarJWTRoundtrip(t *testing.T) {
	token, expiraEm, err := gerarJWT(segredoJWTDemo, "alice", time.Hour)
	if err != nil {
		t.Fatalf("gerarJWT falhou: %v", err)
	}
	if !expiraEm.After(time.Now()) {
		t.Error("expiração deveria ser futura")
	}

	reivindicacoes, err := validarJWT(segredoJWTDemo, token)
	if err != nil {
		t.Fatalf("validarJWT falhou: %v", err)
	}
	if reivindicacoes.Sujeito != "alice" {
		t.Errorf("sujeito esperado alice, obtido %q", reivindicacoes.Sujeito)
	}
	if reivindicacoes.Tipo != TIPO_TOKEN_ACESSO {
		t.Errorf("tipo esperado %q, obtido %q", TIPO_TOKEN_ACESSO, reivindicacoes.Tipo)
	}
}

func TestValidarJWTAssinaturaAdulterada(t *testing.T) {
	token, _, err := gerarJWT(segredoJWTDemo, "alice", time.Hour)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}

	adulterado := adulterarUltimoCaractere(token)
	if _, err := validarJWT(segredoJWTDemo, adulterado); err != ErroJWTInvalido {
		t.Errorf("esperado ErroJWTInvalido, obtido %v", err)
	}
}

func TestValidarJWTSegredoErrado(t *testing.T) {
	token, _, err := gerarJWT(segredoJWTDemo, "alice", time.Hour)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}

	if _, err := validarJWT([]byte("outro-segredo"), token); err != ErroJWTInvalido {
		t.Errorf("esperado ErroJWTInvalido com segredo errado, obtido %v", err)
	}
}

func TestValidarJWTExpirado(t *testing.T) {
	token, _, err := gerarJWT(segredoJWTDemo, "alice", -time.Hour)
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}

	if _, err := validarJWT(segredoJWTDemo, token); err != ErroJWTInvalido {
		t.Errorf("esperado ErroJWTInvalido para token expirado, obtido %v", err)
	}
}

func TestValidarJWTFormatoInvalido(t *testing.T) {
	if _, err := validarJWT(segredoJWTDemo, "nao.é.jwt.valido"); err != ErroJWTInvalido {
		t.Errorf("esperado ErroJWTInvalido para formato inválido, obtido %v", err)
	}
	if _, err := validarJWT(segredoJWTDemo, "soumacoisa"); err != ErroJWTInvalido {
		t.Errorf("esperado ErroJWTInvalido para token sem pontos, obtido %v", err)
	}
}

func TestValidarJWTAlgoritmoNaoSuportado(t *testing.T) {
	// Token corretamente assinado, mas com header alg != HS256: deve ser
	// rejeitado para evitar confusão de algoritmo (ex.: "none").
	cabecalho, err := codificarSegmento(cabecalhoJWT{Algoritmo: "none", Tipo: TIPO_CABECALHO_JWT})
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}
	payload, err := codificarSegmento(ReivindicacoesJWT{
		Sujeito:  "alice",
		ExpiraEm: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("setup falhou: %v", err)
	}
	conteudo := cabecalho + "." + payload
	token := conteudo + "." + assinar(segredoJWTDemo, conteudo)

	if _, err := validarJWT(segredoJWTDemo, token); err != ErroJWTInvalido {
		t.Errorf("esperado ErroJWTInvalido para algoritmo não suportado, obtido %v", err)
	}
}

// ----- Helpers (no final) -----

func adulterarUltimoCaractere(token string) string {
	if token == "" {
		return token
	}
	ultimo := token[len(token)-1]
	substituto := byte('A')
	if ultimo == substituto {
		substituto = 'B'
	}
	return token[:len(token)-1] + string(substituto)
}
