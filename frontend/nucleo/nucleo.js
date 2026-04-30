// ═══════════════════════════════════════════════
//  nucleo/nucleo.js — Orquestrador: troca de secoes, refresh e bootstrap
// ═══════════════════════════════════════════════

(function () {

  // ----- Constantes -----

  const ID_SECAO_PADRAO = 'zabbix';

  const RENDERIZADORES = {
    zabbix:   () => ZabbixSecao.Renderizar(),
    mspcloud: () => MspCloudSecao.Renderizar(),
  };

  const ATUALIZADORES = {
    zabbix:   () => SignalApi.RefreshZabbix(),
    mspcloud: () => SignalApi.RefreshMsp(),
  };


  // ----- Bootstrap -----

  async function inicializar() {
    await renderizarTodas();
    atualizarHorario();
    trocarSecao(ID_SECAO_PADRAO);
    registrarHandlersAbas();
    registrarHandlerAtualizar();
  }


  async function renderizarTodas() {
    await Promise.all(Object.values(RENDERIZADORES).map(fn => fn()));
  }


  function atualizarHorario() {
    const elemento = document.getElementById('ultimaAtualizacao');
    if (!elemento) return;
    const agora = new Date();
    const pad = n => String(n).padStart(2, '0');
    elemento.textContent = `atualizado ${pad(agora.getHours())}:${pad(agora.getMinutes())}:${pad(agora.getSeconds())}`;
  }


  // ----- Troca de secao -----

  function trocarSecao(idSecao) {
    document.querySelectorAll('.secao').forEach(secao => {
      secao.classList.toggle('escondida', secao.id !== `secao-${idSecao}`);
    });
    document.querySelectorAll('.aba').forEach(aba => {
      aba.classList.toggle('ativa', aba.dataset.secao === idSecao);
    });
  }


  // ----- Refresh manual -----

  async function executarRefresh() {
    const botao = document.getElementById('botaoAtualizar');
    if (botao) {
      botao.disabled = true;
      botao.textContent = '⟳ atualizando…';
    }

    try {
      await Promise.all(Object.values(ATUALIZADORES).map(fn => fn().catch(erro => {
        console.warn('Falha em refresh remoto:', erro);
      })));
      await renderizarTodas();
      atualizarHorario();
    } finally {
      if (botao) {
        botao.disabled = false;
        botao.textContent = '⟳ atualizar';
      }
    }
  }


  // ----- Handlers / listeners (sempre no final) -----

  function registrarHandlersAbas() {
    document.querySelectorAll('.aba').forEach(aba => {
      aba.addEventListener('click', () => trocarSecao(aba.dataset.secao));
    });
  }


  function registrarHandlerAtualizar() {
    const botao = document.getElementById('botaoAtualizar');
    if (!botao) return;
    botao.addEventListener('click', executarRefresh);
  }


  document.addEventListener('DOMContentLoaded', inicializar);
})();
