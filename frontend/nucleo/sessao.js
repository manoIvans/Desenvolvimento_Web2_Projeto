// ═══════════════════════════════════════════════
//  nucleo/sessao.js — Estado de sessão no cliente (tokens em localStorage)
//  Fonte única das chaves e da lógica de sessão, compartilhada pelo
//  bootstrap (/), pela página de login e pelo painel.
//
//  Modelo: access token JWT curto (CHAVE_TOKEN, expira em CHAVE_EXPIRA) +
//  refresh token longo (CHAVE_REFRESH). Uma sessão continua "ativa" enquanto
//  houver refresh token, mesmo com o access token expirado — o painel o
//  renova sob demanda.
// ═══════════════════════════════════════════════

const SignalSessao = (function () {

  // ----- Constantes -----

  const CHAVE_TOKEN   = 'signalhubToken';
  const CHAVE_REFRESH = 'signalhubRefresh';
  const CHAVE_EXPIRA  = 'signalhubExpira';


  // ----- Estado da sessão -----

  // Ativa indica que dá para usar a API agora (access válido) ou renovar
  // (há refresh token). É o critério de gate das páginas.
  function Ativa() {
    return AcessoValido() || Renovavel();
  }


  // AcessoValido: há access token e ele ainda não passou da expiração.
  function AcessoValido() {
    const token = localStorage.getItem(CHAVE_TOKEN);
    if (!token) return false;
    const expira = localStorage.getItem(CHAVE_EXPIRA);
    if (!expira) return true;
    return new Date(expira) > new Date();
  }


  // Renovavel: existe refresh token para trocar por um novo par.
  function Renovavel() {
    return !!localStorage.getItem(CHAVE_REFRESH);
  }


  // ----- Persistência -----

  // Guardar salva o par de tokens devolvido por /login ou /refresh.
  function Guardar(dados) {
    if (!dados || !dados.token) return;
    localStorage.setItem(CHAVE_TOKEN, dados.token);
    localStorage.setItem(CHAVE_EXPIRA, dados.expira_em || '');
    if (dados.refresh_token) localStorage.setItem(CHAVE_REFRESH, dados.refresh_token);
  }


  function Limpar() {
    localStorage.removeItem(CHAVE_TOKEN);
    localStorage.removeItem(CHAVE_REFRESH);
    localStorage.removeItem(CHAVE_EXPIRA);
  }


  // ----- Leitura -----

  function Token() {
    return localStorage.getItem(CHAVE_TOKEN);
  }


  function Refresh() {
    return localStorage.getItem(CHAVE_REFRESH);
  }


  return { Ativa, AcessoValido, Renovavel, Guardar, Limpar, Token, Refresh };
})();
