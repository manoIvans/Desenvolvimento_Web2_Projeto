// ═══════════════════════════════════════════════
//  login/scripts/login.js — Página de login do SignalHub
//  Faz POST /login, persiste o token recebido em localStorage
//  e redireciona para /painel/.
// ═══════════════════════════════════════════════

(function () {

  // ----- Constantes -----

  const URL_PAINEL = '/painel/';
  const ROTA_LOGIN = '/login';


  // ----- Bootstrap -----

  function inicializar() {
    if (SignalSessao.Ativa()) {
      window.location.replace(URL_PAINEL);
      return;
    }

    const formulario = document.getElementById('formLogin');
    if (!formulario) return;
    formulario.addEventListener('submit', aoEnviarFormularioHandler);
  }


  // ----- Lógica de login -----

  async function executarLogin(formulario) {
    const senha = lerSenha(formulario);
    if (!senha) {
      exibirMensagem('Informe a senha.', 'erro');
      return;
    }

    const botao = formulario.querySelector('button[type="submit"]');
    desabilitarBotao(botao);

    try {
      const resposta = await chamarApiLogin(senha);
      SignalSessao.Guardar(resposta);
      window.location.replace(URL_PAINEL);
    } catch (erro) {
      exibirMensagem(mensagemDeErro(erro), 'erro');
      restaurarBotao(botao);
    }
  }


  async function chamarApiLogin(senha) {
    const resp = await fetch(ROTA_LOGIN, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept':       'application/json',
      },
      body: JSON.stringify({ senha }),
    });
    if (!resp.ok) {
      const corpo = await resp.json().catch(() => null);
      throw new Error(corpo?.error || `${resp.status} ${resp.statusText}`);
    }
    return await resp.json();
  }


  // ----- Utilitários -----

  function lerSenha(formulario) {
    const campo = formulario.elements.namedItem('senha');
    return campo ? campo.value.trim() : '';
  }


  function exibirMensagem(texto, tipo) {
    const alvo = document.querySelector('[data-mensagem]');
    if (!alvo) return;
    alvo.textContent = texto;
    alvo.className = `login-mensagem ${tipo}`;
  }


  function mensagemDeErro(erro) {
    return erro?.message ?? String(erro);
  }


  function desabilitarBotao(botao) {
    if (!botao) return;
    botao.disabled = true;
    botao.textContent = 'entrando…';
  }


  function restaurarBotao(botao) {
    if (!botao) return;
    botao.disabled = false;
    botao.textContent = 'Entrar';
  }


  // ----- Handlers / listeners (sempre no final) -----

  async function aoEnviarFormularioHandler(evento) {
    evento.preventDefault();
    await executarLogin(evento.target);
  }


  document.addEventListener('DOMContentLoaded', inicializar);
})();
