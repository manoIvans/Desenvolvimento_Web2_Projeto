// ═══════════════════════════════════════════════
//  nucleo/api.js — Wrapper de chamadas HTTP ao backend SignalHub
// ═══════════════════════════════════════════════

const SignalApi = (function () {

  // ----- Constantes -----

  const ROTAS = {
    Zabbix:            '/zabbix',
    ZabbixRefresh:     '/zabbix/refresh',
    Msp:               '/mspclouds',
    MspRefresh:        '/mspclouds/refresh',
    Filtros:           '/filtros',
    InstanciasZabbix:  '/zabbix/instancias',
    InstanciasMsp:     '/mspclouds/instancias',
  };

  const TIMEOUT_REQUISICAO_MS = 20000;


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

  async function chamarAhRota(rota, metodo, corpo) {
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

    try {
      const resp = await fetch(rota, opcoes);
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
  };
})();
