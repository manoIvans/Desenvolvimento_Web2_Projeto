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

  const EVENTO_CONFIGURACOES_ALTERADAS = 'configuracoesAlteradasEvent';

<<<<<<< Updated upstream:frontend/nucleo/nucleo.js
=======
  const URL_LOGIN = '/login/';

>>>>>>> Stashed changes:frontend/painel/nucleo/nucleo.js

  // ----- Bootstrap -----

  async function inicializar() {
<<<<<<< Updated upstream:frontend/nucleo/nucleo.js
=======
    // Sem token → painel não tem o que fazer; volta pro login.
    if (!SignalApi.TokenAtual()) {
      window.location.replace(URL_LOGIN);
      return;
    }

>>>>>>> Stashed changes:frontend/painel/nucleo/nucleo.js
    ConfiguracoesModal.Inicializar();
    await renderizarTodas();
    atualizarHorario();
    trocarSecao(ID_SECAO_PADRAO);
    registrarHandlersAbas();
    registrarHandlerAtualizar();
    registrarHandlerConfiguracoes();
<<<<<<< Updated upstream:frontend/nucleo/nucleo.js
=======
    registrarHandlerSair();
>>>>>>> Stashed changes:frontend/painel/nucleo/nucleo.js
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


  function registrarHandlerConfiguracoes() {
    // Instâncias adicionadas/removidas só refletem no painel após um refresh
    // que reconsulta as fontes — por isso reaproveitamos executarRefresh.
    document.addEventListener(EVENTO_CONFIGURACOES_ALTERADAS, executarRefresh);
  }


<<<<<<< Updated upstream:frontend/nucleo/nucleo.js
=======
  function registrarHandlerSair() {
    const botao = document.getElementById('botaoSair');
    if (!botao) return;
    botao.addEventListener('click', () => SignalApi.Sair());
  }


>>>>>>> Stashed changes:frontend/painel/nucleo/nucleo.js
  document.addEventListener('DOMContentLoaded', inicializar);
})();
