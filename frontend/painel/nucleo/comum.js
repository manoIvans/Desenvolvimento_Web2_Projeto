// ═══════════════════════════════════════════════
//  nucleo/comum.js — Utilitários de renderização compartilhados pelas
//  seções de cards (Zabbix, MSP Clouds, Acronis): escape de HTML,
//  formatação de horário e blocos de estado (carregando/vazio/erro).
// ═══════════════════════════════════════════════

const SignalRender = (function () {

  // ----- Constantes -----

  const TRACO = '—';


  // ----- API pública -----

  function Esqueleto(quantidade) {
    return Array.from({ length: quantidade }, () => '<div class="esqueleto" aria-hidden="true"></div>').join('');
  }


  function BlocoVazio(mensagem) {
    return `<div class="vazio">${Escapar(mensagem)}</div>`;
  }


  function BlocoErro(mensagem) {
    return `<div class="vazio vazio-erro" role="alert">${Escapar(mensagem)}</div>`;
  }


  function FormatarHorario(iso) {
    if (!iso) return TRACO;
    const data = new Date(iso);
    if (isNaN(data.getTime())) return iso;
    const pad = n => String(n).padStart(2, '0');
    return `${data.getFullYear()}-${pad(data.getMonth() + 1)}-${pad(data.getDate())} ` +
           `${pad(data.getHours())}:${pad(data.getMinutes())}`;
  }


  function Escapar(texto) {
    return String(texto ?? '')
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#39;');
  }


  return { Esqueleto, BlocoVazio, BlocoErro, FormatarHorario, Escapar };
})();
