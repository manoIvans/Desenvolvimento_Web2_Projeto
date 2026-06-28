// internal/acronis/tipos.go
//
// Tipos de configuração, de saída (consumidos pelo frontend) e tipos
// internos da API Acronis Cyber Protect Cloud (Cyber Platform API):
// descoberta de datacenter, token OAuth2 e o envelope de alertas.

package acronis

// ----- Configuração -----

// Conta são as credenciais de acesso programático a um tenant Acronis,
// carregadas do ambiente no boot da API.
//
// ServerURL é a URL base do datacenter (ex.: https://eu2-cloud.acronis.com).
// Quando vazia mas Login preenchido, ela é descoberta dinamicamente via
// GET https://cloud.acronis.com/api/1/accounts?login=<Login> — a Acronis
// recomenda jamais fixar a URL do datacenter no código.
type Conta struct {
	ServerURL    string
	Login        string
	ClientID     string
	ClientSecret string
}

// ----- Tipos de saída -----

// Alerta é o formato consolidado consumido pelo frontend, derivado de um
// alerta bruto do Alert Manager da Acronis.
type Alerta struct {
	ID         string         `json:"id"`
	Tipo       string         `json:"tipo"`
	Categoria  string         `json:"categoria,omitempty"`
	Severidade string         `json:"severidade"`
	Titulo     string         `json:"titulo"`
	Descricao  string         `json:"descricao,omitempty"`
	Detalhes   map[string]any `json:"detalhes,omitempty"`
	Horario    string         `json:"horario"`
	Tenant     string         `json:"tenant,omitempty"`
	Source     string         `json:"source,omitempty"`
}

// ResultadoRefresh é o que o endpoint /acronis/refresh devolve.
type ResultadoRefresh struct {
	Data   []Alerta `json:"data"`
	Falhas []string `json:"falhas,omitempty"`
}

// ----- Tipos internos da API Acronis -----

// respostaDescoberta é o corpo de GET /api/1/accounts?login=...
type respostaDescoberta struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	ServerURL string `json:"server_url"`
}

// respostaToken é o corpo de POST /api/2/idp/token. ExpiraEm (expires_on) é
// um timestamp Unix ABSOLUTO em segundos — o instante em que o token deixa
// de valer, não uma duração.
type respostaToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiraEm    int64  `json:"expires_on"`
	IDToken     string `json:"id_token"`
}

// respostaAlertas é o envelope de GET /api/alert_manager/v1/alerts. O objeto
// "paging" da resposta é ignorado de propósito: a API não expõe cursor de
// continuação (a especificação só define os parâmetros `limit` e `skip`),
// então não há próxima página a seguir — o volume é controlado por `limit`.
type respostaAlertas struct {
	Items []alertaRaw `json:"items"`
}

// alertaRaw é um alerta no formato bruto do Alert Manager.
type alertaRaw struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Category string `json:"category"`
	Severity string `json:"severity"`
	Details  struct {
		Title       string         `json:"title"`
		Category    string         `json:"category"`
		Description string         `json:"description"`
		Fields      map[string]any `json:"fields"`
	} `json:"details"`
	CreatedAt  string `json:"createdAt"`
	ReceivedAt string `json:"receivedAt"`
	UpdatedAt  string `json:"updatedAt"`
	Affinity   string `json:"affinity"`
	Tenant     struct {
		ID      string `json:"id"`
		UUID    string `json:"uuid"`
		Locator string `json:"locator"`
	} `json:"tenant"`
}
