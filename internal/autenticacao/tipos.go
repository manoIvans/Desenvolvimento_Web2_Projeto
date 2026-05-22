// internal/autenticacao/tipos.go
//
// DTOs do domínio Autenticação: entrada do POST /login e o envelope de
// resposta com o token temporário gerado.

package autenticacao

// ----- Tipo de entrada -----

// EntradaLogin é o corpo aceito em POST /login.
type EntradaLogin struct {
	Senha string `json:"senha"`
}

// ----- Tipo de saída -----

// RespostaLogin é o envelope devolvido por POST /login: token aleatório
// e o momento exato em que ele expira (RFC 3339, UTC).
type RespostaLogin struct {
	Token    string `json:"token"`
	ExpiraEm string `json:"expira_em"`
}
