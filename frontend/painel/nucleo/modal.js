// ═══════════════════════════════════════════════
//  nucleo/modal.js — Mecânica compartilhada de modal (abrir/fechar com
//  gestão de foco, Escape, focus trap e clique no backdrop) e helpers de
//  formulário/lista reusados pelos modais de configurações e de filtros.
// ═══════════════════════════════════════════════

const SignalModal = (function () {

  // ----- Constantes -----

  const SELETOR_FOCAVEIS =
    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';


  // ----- Controlador de modal -----

  // Criar liga um overlay a um controlador com abrir/fechar. Cuida do foco
  // (move para dentro ao abrir, restaura ao fechar), de Escape, do focus trap
  // (Tab/Shift+Tab) e do clique no backdrop. aoFechar (opcional) roda antes de
  // restaurar o foco — útil para emitir um evento de "houve alteração".
  function Criar({ overlayId, aoFechar }) {
    let focoAnterior = null;

    function overlay() {
      return document.getElementById(overlayId);
    }

    function estaAberto() {
      const el = overlay();
      return !!el && !el.classList.contains('escondida');
    }

    function abrir(focoInicial) {
      const el = overlay();
      if (!el) return;
      focoAnterior = document.activeElement;
      el.classList.remove('escondida');
      if (focoInicial && typeof focoInicial.focus === 'function') focoInicial.focus();
    }

    function fechar() {
      const el = overlay();
      if (!el) return;
      el.classList.add('escondida');
      if (typeof aoFechar === 'function') aoFechar();
      if (focoAnterior && typeof focoAnterior.focus === 'function') focoAnterior.focus();
      focoAnterior = null;
    }

    function aoTeclarHandler(evento) {
      if (!estaAberto()) return;
      if (evento.key === 'Escape') {
        fechar();
        return;
      }
      if (evento.key === 'Tab') {
        prenderFoco(evento, overlay());
      }
    }

    function aoClicarBackdropHandler(evento) {
      if (evento.target.id === overlayId) fechar();
    }

    const el = overlay();
    if (el) el.addEventListener('click', aoClicarBackdropHandler);
    document.addEventListener('keydown', aoTeclarHandler);

    return { abrir, fechar, estaAberto };
  }


  // prenderFoco mantém o Tab/Shift+Tab ciclando dentro do diálogo (focus trap).
  function prenderFoco(evento, overlay) {
    if (!overlay) return;
    const focaveis = overlay.querySelectorAll(SELETOR_FOCAVEIS);
    const visiveis = Array.from(focaveis).filter(el => !el.disabled && el.offsetParent !== null);
    if (visiveis.length === 0) return;

    const primeiro = visiveis[0];
    const ultimo = visiveis[visiveis.length - 1];

    if (evento.shiftKey && document.activeElement === primeiro) {
      evento.preventDefault();
      ultimo.focus();
    } else if (!evento.shiftKey && document.activeElement === ultimo) {
      evento.preventDefault();
      primeiro.focus();
    }
  }


  // ----- Helpers de formulário/lista -----

  // BotoesForm devolve os botões de ação do formulário (Adicionar/Salvar +
  // Cancelar no modo edição). emEdicao truthy ⇒ modo edição.
  function BotoesForm(emEdicao) {
    const cancelar = emEdicao
      ? `<button type="button" class="config-botao secundario" data-acao="cancelar">Cancelar</button>`
      : '';
    return `
      <div class="config-form-acoes">
        <button type="submit" class="config-botao">${emEdicao ? 'Salvar alterações' : 'Adicionar'}</button>
        ${cancelar}
      </div>`;
  }


  function BotoesItem(id) {
    return `
      <div class="config-item-acoes">
        <button type="button" class="config-botao secundario" data-acao="editar" data-id="${id}">Editar</button>
        <button type="button" class="config-botao perigo"    data-acao="remover" data-id="${id}">Remover</button>
      </div>`;
  }


  function ItemVazio(texto) {
    return `<li class="config-item-vazio">${SignalRender.Escapar(texto)}</li>`;
  }


  function ValorCampo(formulario, nome) {
    const campo = formulario.elements.namedItem(nome);
    return campo ? campo.value.trim() : '';
  }


  // ExibirMensagem escreve um feedback no [data-mensagem] dentro do escopo
  // informado (o container de conteúdo do modal).
  function ExibirMensagem(escopo, texto, tipo) {
    const alvo = escopo?.querySelector('[data-mensagem]');
    if (!alvo) return;
    alvo.textContent = texto;
    alvo.className = `config-mensagem ${tipo}`;
  }


  function MensagemDeErro(erro) {
    return erro?.message ?? String(erro);
  }


  return {
    Criar,
    BotoesForm, BotoesItem, ItemVazio, ValorCampo, ExibirMensagem, MensagemDeErro,
  };
})();
