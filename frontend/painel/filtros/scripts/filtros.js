// ═══════════════════════════════════════════════
//  filtros/scripts/filtros.js — Janela dedicada de filtros do Zabbix
//  (CRUD em /filtros). Filtros são exclusivos do Zabbix; por isso têm um
//  pop-up próprio, aberto pelo botão no cabeçalho da seção Zabbix.
// ═══════════════════════════════════════════════

const FiltrosModal = (function () {

  // ----- Constantes -----

  const EVENTO_DADOS_ALTERADOS = 'configuracoesAlteradasEvent';

  const ROTULOS_ALVO = {
    hosts:            'Hosts',
    eventos:          'Eventos',
    grupos:           'Grupos',
    eventos_em_hosts: 'Eventos em Hosts',
  };

  // Cada alvo de filtro usa apenas os campos que fazem sentido para ele.
  const CAMPOS_POR_ALVO = {
    hosts:            ['host'],
    eventos:          ['evento'],
    grupos:           ['valor'],
    eventos_em_hosts: ['evento', 'host'],
  };

  const ROTULOS_CAMPO_FILTRO = {
    valor:  'Grupo',
    evento: 'Evento',
    host:   'Host',
  };

  const ALVO_PADRAO  = 'hosts';
  const ID_CONTEUDO  = 'modalFiltrosConteudo';


  // ----- Estado -----

  let idEmEdicao = null;
  let houveAlteracao = false;
  let cacheInstanciasZabbix = [];

  const escapar = SignalRender.Escapar;

  const modal = SignalModal.Criar({
    overlayId: 'modalFiltros',
    aoFechar: () => {
      if (houveAlteracao) document.dispatchEvent(new CustomEvent(EVENTO_DADOS_ALTERADOS));
    },
  });


  // ----- API pública -----

  function Inicializar() {
    registrarHandlers();
  }


  // ----- Controle do modal -----

  function abrir() {
    idEmEdicao = null;
    houveAlteracao = false;
    modal.abrir(document.getElementById('botaoFecharFiltros'));
    renderizar();
  }


  // ----- Render -----

  async function renderizar() {
    const conteudo = document.getElementById(ID_CONTEUDO);
    if (!conteudo) return;
    conteudo.innerHTML = `<div class="config-carregando">Carregando…</div>`;

    try {
      await renderizarFiltros(conteudo);
    } catch (erro) {
      conteudo.innerHTML = `<div class="config-mensagem erro">${escapar(SignalModal.MensagemDeErro(erro))}</div>`;
    }
  }


  async function renderizarFiltros(conteudo) {
    const [respostaFiltros, respostaInstancias] = await Promise.all([
      SignalApi.ListarFiltros(),
      SignalApi.ListarInstanciasZabbix(),
    ]);
    const filtros = respostaFiltros?.data ?? [];
    cacheInstanciasZabbix = respostaInstancias?.data ?? [];

    if (cacheInstanciasZabbix.length === 0) {
      conteudo.innerHTML = `
        <div class="config-mensagem aviso">
          Cadastre ao menos uma instância Zabbix antes de criar filtros.
        </div>`;
      return;
    }

    const emEdicao = filtros.find(f => f.id === idEmEdicao);
    const alvoAtual = emEdicao?.alvo ?? ALVO_PADRAO;
    conteudo.innerHTML = `
      <form class="config-form" data-form="filtros">
        <h3>${idEmEdicao ? 'Editar filtro' : 'Novo filtro'}</h3>
        <label>Instância Zabbix *
          <select name="instancia_id" ${idEmEdicao ? 'disabled' : ''} required>
            ${montarOpcoesInstancia(emEdicao?.instancia_id)}
          </select>
        </label>
        <label>Alvo *
          <select name="alvo" required>${montarOpcoesAlvo(alvoAtual)}</select>
        </label>
        <div id="camposFiltro">${montarCamposFiltro(alvoAtual, emEdicao ?? {})}</div>
        ${SignalModal.BotoesForm(idEmEdicao)}
      </form>
      <div class="config-mensagem" data-mensagem role="alert" aria-live="polite"></div>
      <ul class="config-lista">
        ${filtros.map(montarLinhaFiltro).join('') || SignalModal.ItemVazio('Nenhum filtro cadastrado.')}
      </ul>
    `;
  }


  // montarCamposFiltro gera apenas os inputs relevantes para o alvo
  // selecionado. Campo único vira obrigatório; com dois, ao menos um.
  function montarCamposFiltro(alvo, valores) {
    const nomes = CAMPOS_POR_ALVO[alvo] ?? [];
    const obrigatorioUnico = nomes.length === 1;

    const dica = nomes.length > 1
      ? `<p class="config-dica">Preencha ao menos um dos campos abaixo.</p>`
      : '';

    const campos = nomes.map(nome => `
      <label>${ROTULOS_CAMPO_FILTRO[nome]}
        <input name="${nome}" type="text" maxlength="200"
               ${obrigatorioUnico ? 'required' : ''}
               value="${escapar(valores[nome] ?? '')}" />
      </label>
    `).join('');

    return dica + campos;
  }


  function montarLinhaFiltro(filtro) {
    const partes = [];
    if (filtro.valor)  partes.push(`valor: ${filtro.valor}`);
    if (filtro.evento) partes.push(`evento: ${filtro.evento}`);
    if (filtro.host)   partes.push(`host: ${filtro.host}`);

    return `
      <li class="config-item">
        <div class="config-item-info">
          <span class="config-item-titulo">${escapar(ROTULOS_ALVO[filtro.alvo] ?? filtro.alvo)}</span>
          <span class="config-item-detalhe">${escapar(nomeDaInstancia(filtro.instancia_id))}</span>
          <span class="config-item-detalhe">${escapar(partes.join(' · ') || '—')}</span>
        </div>
        ${SignalModal.BotoesItem(filtro.id)}
      </li>
    `;
  }


  function montarOpcoesInstancia(selecionada) {
    return cacheInstanciasZabbix.map(instancia => {
      const marca = instancia.id === selecionada ? 'selected' : '';
      const rotulo = instancia.nome || instancia.url;
      return `<option value="${instancia.id}" ${marca}>${escapar(rotulo)}</option>`;
    }).join('');
  }


  function montarOpcoesAlvo(selecionado) {
    return Object.entries(ROTULOS_ALVO).map(([valor, rotulo]) => {
      const marca = valor === selecionado ? 'selected' : '';
      return `<option value="${valor}" ${marca}>${escapar(rotulo)}</option>`;
    }).join('');
  }


  function nomeDaInstancia(id) {
    const instancia = cacheInstanciasZabbix.find(i => i.id === id);
    if (!instancia) return `instância #${id}`;
    return instancia.nome || instancia.url;
  }


  // ----- Ações -----

  async function salvarFiltro(formulario) {
    const dados = {
      instancia_id: Number(SignalModal.ValorCampo(formulario, 'instancia_id')),
      alvo:         SignalModal.ValorCampo(formulario, 'alvo'),
      valor:        SignalModal.ValorCampo(formulario, 'valor'),
      evento:       SignalModal.ValorCampo(formulario, 'evento'),
      host:         SignalModal.ValorCampo(formulario, 'host'),
    };
    if (idEmEdicao) {
      await SignalApi.AtualizarFiltro(idEmEdicao, dados);
    } else {
      await SignalApi.CriarFiltro(dados);
    }
  }


  function exibirMensagem(texto, tipo) {
    SignalModal.ExibirMensagem(document.getElementById(ID_CONTEUDO), texto, tipo);
  }


  // ----- Handlers / listeners (sempre no final) -----

  function registrarHandlers() {
    const botaoAbrir = document.getElementById('botaoFiltrosZabbix');
    if (botaoAbrir) botaoAbrir.addEventListener('click', abrir);

    const botaoFechar = document.getElementById('botaoFecharFiltros');
    if (botaoFechar) botaoFechar.addEventListener('click', () => modal.fechar());

    const conteudo = document.getElementById(ID_CONTEUDO);
    if (conteudo) {
      conteudo.addEventListener('submit', aoEnviarFormularioHandler);
      conteudo.addEventListener('click', aoClicarConteudoHandler);
      conteudo.addEventListener('change', aoMudarAlvoHandler);
    }
  }


  async function aoEnviarFormularioHandler(evento) {
    evento.preventDefault();
    const formulario = evento.target;

    try {
      await salvarFiltro(formulario);
      houveAlteracao = true;
      idEmEdicao = null;
      await renderizar();
      exibirMensagem('Filtro salvo com sucesso.', 'sucesso');
    } catch (erro) {
      exibirMensagem(SignalModal.MensagemDeErro(erro), 'erro');
    }
  }


  function aoMudarAlvoHandler(evento) {
    if (evento.target.name !== 'alvo') return;

    const container = document.getElementById('camposFiltro');
    const formulario = evento.target.form;
    if (!container || !formulario) return;

    // Preserva o que já foi digitado nos campos que continuam visíveis.
    const valores = {
      valor:  SignalModal.ValorCampo(formulario, 'valor'),
      evento: SignalModal.ValorCampo(formulario, 'evento'),
      host:   SignalModal.ValorCampo(formulario, 'host'),
    };
    container.innerHTML = montarCamposFiltro(evento.target.value, valores);
  }


  async function aoClicarConteudoHandler(evento) {
    const botao = evento.target.closest('[data-acao]');
    if (!botao) return;

    const acao = botao.dataset.acao;
    if (acao === 'cancelar') {
      idEmEdicao = null;
      await renderizar();
      return;
    }

    const id = Number(botao.dataset.id);
    if (acao === 'editar') {
      idEmEdicao = id;
      await renderizar();
      return;
    }

    if (acao === 'remover') {
      await removerComConfirmacao(id);
    }
  }


  async function removerComConfirmacao(id) {
    if (!window.confirm('Remover este filtro? Esta ação não pode ser desfeita.')) return;

    try {
      await SignalApi.RemoverFiltro(id);
      houveAlteracao = true;
      idEmEdicao = null;
      await renderizar();
      exibirMensagem('Filtro removido.', 'sucesso');
    } catch (erro) {
      exibirMensagem(SignalModal.MensagemDeErro(erro), 'erro');
    }
  }


  return { Inicializar };
})();
