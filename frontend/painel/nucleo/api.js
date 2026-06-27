// ═══════════════════════════════════════════════
//  nucleo/api.js — Wrapper de chamadas HTTP ao backend SignalHub
//  · Injeta Authorization: Bearer <token> nas rotas protegidas
//  · Em 401, tenta renovar a sessão (refresh token) e repete a chamada;
//    se a renovação falhar, limpa a sessão e redireciona para /login/
// ═══════════════════════════════════════════════

const SignalApi = (function () {

  // ----- Constantes -----

  const ROTAS = {
    Zabbix:           '/zabbix',
    ZabbixRefresh:    '/zabbix/refresh',
    Msp:              '/mspclouds',
    MspRefresh:       '/mspclouds/refresh',
    Filtros:          '/filtros',
    InstanciasZabbix: '/zabbix/instancias',
    InstanciasMsp:    '/mspclouds/instancias',
  };

  const URL_LOGIN    = '/login/';
  const ROTA_REFRESH = '/refresh';
  const ROTA_LOGOUT  = '/logout';

  const TIMEOUT_REQUISICAO_MS = 20000;

  // Renovação em curso — coalesce 401s simultâneos para que apenas um
  // /refresh seja disparado. O refresh token é de uso único (rotacionado a
  // cada troca); chamadas paralelas com o mesmo token invalidariam umas às
  // outras e derrubariam a sessão.
  let renovacaoEmAndamento = null;


  // ----- Sessão -----

  function TokenAtual() {
    return SignalSessao.Token();
  }


  function Sair() {
    revogarRefreshRemoto();
    SignalSessao.Limpar();
    if (window.location.pathname.startsWith('/login')) return;
    window.location.replace(URL_LOGIN);
  }


  // ----- Renovação de sessão (refresh token) -----

  // renovarSessao garante uma única renovação concorrente: chamadas
  // simultâneas aguardam a mesma promessa em vez de gastar o refresh token.
  async function renovarSessao() {
    if (!renovacaoEmAndamento) {
      renovacaoEmAndamento = executarRenovacao().finally(() => {
        renovacaoEmAndamento = null;
      });
    }
    return await renovacaoEmAndamento;
  }


  async function executarRenovacao() {
    const refresh = SignalSessao.Refresh();
    if (!refresh) return false;

    try {
      const resp = await fetch(ROTA_REFRESH, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
        body: JSON.stringify({ refresh_token: refresh }),
      });
      if (!resp.ok) return false;
      SignalSessao.Guardar(await resp.json());
      return true;
    } catch (_) {
      return false;
    }
  }


  function revogarRefreshRemoto() {
    const refresh = SignalSessao.Refresh();
    if (!refresh) return;
    // keepalive garante o envio mesmo durante a navegação que segue o logout.
    fetch(ROTA_LOGOUT, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refresh }),
      keepalive: true,
    }).catch(() => {});
  }


  // ----- Painel: alertas Zabbix / MSP -----

  async function BuscarZabbix() {
    return await chamarAhRota(ROTAS.Zabbix, 'GET');
  }


  async function RefreshZabbix() {
    return await chamarAhRota(ROTAS.ZabbixRefresh, 'POST');
  }


  async function BuscarMsp() {
    return await chamarAhRota(ROTAS.Msp, 'GET');
  }


  async function RefreshMsp() {
    return await chamarAhRota(ROTAS.MspRefresh, 'POST');
  }


  // ----- Filtros (exclusivo Zabbix) -----

  async function ListarFiltros() {
    return await chamarAhRota(ROTAS.Filtros, 'GET');
  }


  async function CriarFiltro(dados) {
    return await chamarAhRota(ROTAS.Filtros, 'POST', dados);
  }


  async function AtualizarFiltro(id, dados) {
    return await chamarAhRota(`${ROTAS.Filtros}/${id}`, 'PUT', dados);
  }


  async function RemoverFiltro(id) {
    return await chamarAhRota(`${ROTAS.Filtros}/${id}`, 'DELETE');
  }


  // ----- Instâncias Zabbix -----

  async function ListarInstanciasZabbix() {
    return await chamarAhRota(ROTAS.InstanciasZabbix, 'GET');
  }


  async function CriarInstanciaZabbix(dados) {
    return await chamarAhRota(ROTAS.InstanciasZabbix, 'POST', dados);
  }


  async function AtualizarInstanciaZabbix(id, dados) {
    return await chamarAhRota(`${ROTAS.InstanciasZabbix}/${id}`, 'PUT', dados);
  }


  async function RemoverInstanciaZabbix(id) {
    return await chamarAhRota(`${ROTAS.InstanciasZabbix}/${id}`, 'DELETE');
  }


  // ----- Instâncias MSP Clouds -----

  async function ListarInstanciasMsp() {
    return await chamarAhRota(ROTAS.InstanciasMsp, 'GET');
  }


  async function CriarInstanciaMsp(dados) {
    return await chamarAhRota(ROTAS.InstanciasMsp, 'POST', dados);
  }


  async function AtualizarInstanciaMsp(id, dados) {
    return await chamarAhRota(`${ROTAS.InstanciasMsp}/${id}`, 'PUT', dados);
  }


  async function RemoverInstanciaMsp(id) {
    return await chamarAhRota(`${ROTAS.InstanciasMsp}/${id}`, 'DELETE');
  }


  // ----- Internos -----

  async function chamarAhRota(rota, metodo, corpo, jaRenovou = false) {
    const controlador = new AbortController();
    const idTimeout = setTimeout(() => controlador.abort(), TIMEOUT_REQUISICAO_MS);

    const opcoes = {
      method: metodo,
      headers: { 'Accept': 'application/json' },
      signal: controlador.signal,
    };
    if (corpo !== undefined) {
      opcoes.headers['Content-Type'] = 'application/json';
      opcoes.body = JSON.stringify(corpo);
    }

    const token = TokenAtual();
    if (token) opcoes.headers['Authorization'] = `Bearer ${token}`;

    try {
      const resp = await fetch(rota, opcoes);
      if (resp.status === 401) {
        if (!jaRenovou && await renovarSessao()) {
          return await chamarAhRota(rota, metodo, corpo, true);
        }
        Sair();
        throw new Error('Sessão expirada. Faça login novamente.');
      }
      if (!resp.ok) {
        throw new Error(await lerMensagemErro(resp, metodo, rota));
      }
      if (resp.status === 204) return null;
      return await resp.json();
    } finally {
      clearTimeout(idTimeout);
    }
  }


  async function lerMensagemErro(resp, metodo, rota) {
    try {
      const corpo = await resp.json();
      if (corpo && corpo.error) return corpo.error;
    } catch (_) {
      // corpo não-JSON — cai no formato genérico abaixo
    }
    return `${metodo} ${rota} → ${resp.status} ${resp.statusText}`;
  }


  return {
    BuscarZabbix, RefreshZabbix, BuscarMsp, RefreshMsp,
    ListarFiltros, CriarFiltro, AtualizarFiltro, RemoverFiltro,
    ListarInstanciasZabbix, CriarInstanciaZabbix, AtualizarInstanciaZabbix, RemoverInstanciaZabbix,
    ListarInstanciasMsp, CriarInstanciaMsp, AtualizarInstanciaMsp, RemoverInstanciaMsp,
    TokenAtual, Sair,
  };
})();
