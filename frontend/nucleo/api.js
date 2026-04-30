// ═══════════════════════════════════════════════
//  nucleo/api.js — Wrapper de chamadas HTTP ao backend SignalHub
// ═══════════════════════════════════════════════

const SignalApi = (function () {

  // ----- Constantes -----

  const ROTAS = {
    Zabbix:        '/zabbix',
    ZabbixRefresh: '/zabbix/refresh',
    Msp:           '/mspclouds',
    MspRefresh:    '/mspclouds/refresh',
  };

  const TIMEOUT_REQUISICAO_MS = 20000;


  // ----- API publica -----

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


  // ----- Internos -----

  async function chamarAhRota(rota, metodo) {
    const controlador = new AbortController();
    const idTimeout = setTimeout(() => controlador.abort(), TIMEOUT_REQUISICAO_MS);

    try {
      const resp = await fetch(rota, {
        method: metodo,
        headers: { 'Accept': 'application/json' },
        signal: controlador.signal,
      });
      if (!resp.ok) {
        const corpo = await resp.text();
        throw new Error(`${metodo} ${rota} → ${resp.status}: ${corpo || resp.statusText}`);
      }
      return await resp.json();
    } finally {
      clearTimeout(idTimeout);
    }
  }


  return { BuscarZabbix, RefreshZabbix, BuscarMsp, RefreshMsp };
})();
