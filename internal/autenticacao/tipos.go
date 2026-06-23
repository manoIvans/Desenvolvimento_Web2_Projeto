// internal/autenticacao/tipos.go
//
// DTOs do domínio Autenticação: corpos aceitos em /login, /refresh e
// /logout, e o envelope de resposta com o par de tokens emitido.

package autenticacao

// ----- Tipos de entrada -----

// EntradaLogin é o corpo aceito em POST /login.
type EntradaLogin struct {
	Senha string `json:"senha"`
}

// EntradaRefresh é o corpo aceito em POST /refresh e POST /logout.
type EntradaRefresh struct {
	TokenRefresh string `json:"refresh_token"`
}

// ----- Tipo de saída -----

// RespostaSessao é o envelope devolvido por /login e /refresh: access token
// JWT, refresh token opaco, o instante de expiração do access (RFC 3339,
// UTC) e o esquema de autorização a usar no header (Bearer).
type RespostaSessao struct {
	Token        string `json:"token"`
	TokenRefresh string `json:"refresh_token"`
	ExpiraEm     string `json:"expira_em"`
	Tipo         string `json:"tipo"`
}
