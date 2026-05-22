// ═══════════════════════════════════════════════
//  configuracoes/scripts/configuracoes.js
//  Modal de configurações — CRUD de instâncias Zabbix, instâncias MSP
//  Clouds e filtros (exclusivos do Zabbix), persistidos no backend.
// ═══════════════════════════════════════════════

const ConfiguracoesModal = (function () {

  // ----- Constantes -----

  const PAINEL_PADRAO = 'zabbix';

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

  const ALVO_PADRAO = 'hosts';

  const PAINEIS = {
    zabbix:   () => renderizarPainelZabbix(),
    mspcloud: () => renderizarPainelMsp(),
    filtros:  () => renderizarPainelFiltros(),
  };


  // ----- Estado -----

  let painelAtivo = PAINEL_PADRAO;
  let idEmEdicao  = null;
  let houveAlteracao = false;

  let cacheInstanciasZabbix = [];


  // ----- API pública -----

  function Inicializar() {
    registrarHandlers();
  }


  // ----- Controle do modal -----

  function abrir() {
    const overlay = document.getElementById('modalConfiguracoes');
    if (!overlay) return;
    overlay.classList.remove('escondida');
    houveAlteracao = false;
    trocarPainel(PAINEL_PADRAO);
  }


  function fechar() {
    const overlay = document.getElementById('modalConfiguracoes');
    if (!overlay) return;
    overlay.classList.add('escondida');
    if (houveAlteracao) {
      document.dispatchEvent(new CustomEvent(EVENTO_DADOS_ALTERADOS));
    }
  }


  function trocarPainel(idPainel) {
    painelAtivo = idPainel;
    idEmEdicao  = null;
    document.querySelectorAll('.modal-aba').forEach(aba => {
      aba.classList.toggle('ativa', aba.dataset.painel === idPainel);
    });
    renderizarPainelAtivo();
  }


  async function renderizarPainelAtivo() {
    const conteudo = document.getElementById('modalConteudo');
    if (!conteudo) return;
    conteudo.innerHTML = `<div class="config-carregando">Carregando…</div>`;

    const renderizador = PAINEIS[painelAtivo];
    if (!renderizador) return;

    try {
      await renderizador();
    } catch (erro) {
      conteudo.innerHTML = `<div class="config-mensagem erro">${escapar(mensagemDeErro(erro))}</div>`;
    }
  }


  // ----- Painel: instâncias Zabbix -----

  async function renderizarPainelZabbix() {
    const resposta = await SignalApi.ListarInstanciasZabbix();
    const lista = resposta?.data ?? [];
    const emEdicao = lista.find(i => i.id === idEmEdicao);

    document.getElementById('modalConteudo').innerHTML = `
      <form class="config-form" data-form="zabbix">
        <h3>${idEmEdicao ? 'Editar instância Zabbix' : 'Nova instância Zabbix'}</h3>
        <label>Nome
          <input name="nome" type="text" maxlength="120"
                 value="${escapar(emEdicao?.nome ?? '')}" placeholder="Instância de produção" />
        </label>
        <label>URL da API JSON-RPC *
          <input name="url" type="url" maxlength="500" required
                 value="${escapar(emEdicao?.url ?? '')}"
                 placeholder="https://zabbix.exemplo.com/api_jsonrpc.php" />
        </label>
        <label>API Key *
          <input name="api_key" type="text" maxlength="500" required
                 value="${escapar(emEdicao?.api_key ?? '')}" placeholder="token de acesso" />
        </label>
        ${montarBotoesForm()}
      </form>
      <div class="config-mensagem" data-mensagem></div>
      <ul class="config-lista">
        ${lista.map(montarLinhaZabbix).join('') || itemVazio('Nenhuma instância Zabbix cadastrada.')}
      </ul>
    `;
  }


  function montarLinhaZabbix(instancia) {
    return `
      <li class="config-item">
        <div class="config-item-info">
          <span class="config-item-titulo">${escapar(instancia.nome || 'sem nome')}</span>
          <span class="config-item-detalhe">${escapar(instancia.url)}</span>
          <span class="config-item-detalhe">chave: ${escapar(mascarar(instancia.api_key))}</span>
        </div>
        ${montarBotoesItem(instancia.id)}
      </li>
    `;
  }


  // ----- Painel: instâncias MSP Clouds -----

  async function renderizarPainelMsp() {
    const resposta = await SignalApi.ListarInstanciasMsp();
    const lista = resposta?.data ?? [];
    const emEdicao = lista.find(i => i.id === idEmEdicao);

    document.getElementById('modalConteudo').innerHTML = `
      <form class="config-form" data-form="mspcloud">
        <h3>${idEmEdicao ? 'Editar instância MSP Clouds' : 'Nova instância MSP Clouds'}</h3>
        <label>API Key *
          <input name="api_key" type="text" maxlength="500" required
                 value="${escapar(emEdicao?.api_key ?? '')}" placeholder="2058-0A8B-6FE0-8480" />
        </label>
        ${montarBotoesForm()}
      </form>
      <div class="config-mensagem" data-mensagem></div>
      <ul class="config-lista">
        ${lista.map(montarLinhaMsp).join('') || itemVazio('Nenhuma instância MSP Clouds cadastrada.')}
      </ul>
    `;
  }


  function montarLinhaMsp(instancia) {
    return `
      <li class="config-item">
        <div class="config-item-info">
          <span class="config-item-titulo">${escapar(mascarar(instancia.api_key))}</span>
          <span class="config-item-detalhe">cadastrada em ${escapar(formatarData(instancia.criado_em))}</span>
        </div>
        ${montarBotoesItem(instancia.id)}
      </li>
    `;
  }


  // ----- Painel: filtros (exclusivo Zabbix) -----

  async function renderizarPainelFiltros() {
    const [respostaFiltros, respostaInstancias] = await Promise.all([
      SignalApi.ListarFiltros(),
      SignalApi.ListarInstanciasZabbix(),
    ]);
    const filtros = respostaFiltros?.data ?? [];
    cacheInstanciasZabbix = respostaInstancias?.data ?? [];

    if (cacheInstanciasZabbix.length === 0) {
      document.getElementById('modalConteudo').innerHTML = `
        <div class="config-mensagem aviso">
          Cadastre ao menos uma instância Zabbix antes de criar filtros.
        </div>`;
      return;
    }

    const emEdicao = filtros.find(f => f.id === idEmEdicao);
    const alvoAtual = emEdicao?.alvo ?? ALVO_PADRAO;
    document.getElementById('modalConteudo').innerHTML = `
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
        ${montarBotoesForm()}
      </form>
      <div class="config-mensagem" data-mensagem></div>
      <ul class="config-lista">
        ${filtros.map(montarLinhaFiltro).join('') || itemVazio('Nenhum filtro cadastrado.')}
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
        ${montarBotoesItem(filtro.id)}
      </li>
    `;
  }


  // ----- Ações: salvar -----

  async function salvarPainelAtivo(formulario) {
    if (painelAtivo === 'zabbix')   return await salvarZabbix(formulario);
    if (painelAtivo === 'mspcloud') return await salvarMsp(formulario);
    return await salvarFiltro(formulario);
  }


  async function salvarZabbix(formulario) {
    const dados = {
      nome:    valorCampo(formulario, 'nome'),
      url:     valorCampo(formulario, 'url'),
      api_key: valorCampo(formulario, 'api_key'),
    };
    if (idEmEdicao) {
      await SignalApi.AtualizarInstanciaZabbix(idEmEdicao, dados);
    } else {
      await SignalApi.CriarInstanciaZabbix(dados);
    }
  }


  async function salvarMsp(formulario) {
    const dados = { api_key: valorCampo(formulario, 'api_key') };
    if (idEmEdicao) {
      await SignalApi.AtualizarInstanciaMsp(idEmEdicao, dados);
    } else {
      await SignalApi.CriarInstanciaMsp(dados);
    }
  }


  async function salvarFiltro(formulario) {
    const dados = {
      instancia_id: Number(valorCampo(formulario, 'instancia_id')),
      alvo:         valorCampo(formulario, 'alvo'),
      valor:        valorCampo(formulario, 'valor'),
      evento:       valorCampo(formulario, 'evento'),
      host:         valorCampo(formulario, 'host'),
    };
    if (idEmEdicao) {
      await SignalApi.AtualizarFiltro(idEmEdicao, dados);
    } else {
      await SignalApi.CriarFiltro(dados);
    }
  }


  // ----- Ações: remover -----

  async function removerPainelAtivo(id) {
    if (painelAtivo === 'zabbix')   return await SignalApi.RemoverInstanciaZabbix(id);
    if (painelAtivo === 'mspcloud') return await SignalApi.RemoverInstanciaMsp(id);
    return await SignalApi.RemoverFiltro(id);
  }


  // ----- Utilitários de montagem -----

  function montarBotoesForm() {
    const cancelar = idEmEdicao
      ? `<button type="button" class="config-botao secundario" data-acao="cancelar">Cancelar</button>`
      : '';
    return `
      <div class="config-form-acoes">
        <button type="submit" class="config-botao">${idEmEdicao ? 'Salvar alterações' : 'Adicionar'}</button>
        ${cancelar}
      </div>`;
  }


  function montarBotoesItem(id) {
    return `
      <div class="config-item-acoes">
        <button type="button" class="config-botao secundario" data-acao="editar" data-id="${id}">Editar</button>
        <button type="button" class="config-botao perigo"    data-acao="remover" data-id="${id}">Remover</button>
      </div>`;
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


  function itemVazio(texto) {
    return `<li class="config-item-vazio">${escapar(texto)}</li>`;
  }


  // ----- Utilitários gerais -----

  function valorCampo(formulario, nome) {
    const campo = formulario.elements.namedItem(nome);
    return campo ? campo.value.trim() : '';
  }


  function nomeDaInstancia(id) {
    const instancia = cacheInstanciasZabbix.find(i => i.id === id);
    if (!instancia) return `instância #${id}`;
    return instancia.nome || instancia.url;
  }


  function exibirMensagem(texto, tipo) {
    const alvo = document.querySelector('#modalConteudo [data-mensagem]');
    if (!alvo) return;
    alvo.textContent = texto;
    alvo.className = `config-mensagem ${tipo}`;
  }


  function mensagemDeErro(erro) {
    return erro?.message ?? String(erro);
  }


  function mascarar(chave) {
    if (!chave || chave.length <= 6) return '••••';
    return `${chave.slice(0, 6)}…`;
  }


  function formatarData(iso) {
    if (!iso) return '—';
    const data = new Date(iso);
    if (isNaN(data.getTime())) return iso;
    const pad = n => String(n).padStart(2, '0');
    return `${data.getFullYear()}-${pad(data.getMonth() + 1)}-${pad(data.getDate())}`;
  }


  function escapar(texto) {
    return String(texto ?? '')
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#39;');
  }


  // ----- Handlers / listeners (sempre no final) -----

  function registrarHandlers() {
    const botaoAbrir = document.getElementById('botaoConfiguracoes');
    if (botaoAbrir) botaoAbrir.addEventListener('click', abrir);

    const botaoFechar = document.getElementById('botaoFecharConfiguracoes');
    if (botaoFechar) botaoFechar.addEventListener('click', fechar);

    const overlay = document.getElementById('modalConfiguracoes');
    if (overlay) overlay.addEventListener('click', aoClicarOverlayHandler);

    document.querySelectorAll('.modal-aba').forEach(aba => {
      aba.addEventListener('click', () => trocarPainel(aba.dataset.painel));
    });

    const conteudo = document.getElementById('modalConteudo');
    if (conteudo) {
      conteudo.addEventListener('submit', aoEnviarFormularioHandler);
      conteudo.addEventListener('click', aoClicarConteudoHandler);
      conteudo.addEventListener('change', aoMudarAlvoHandler);
    }

    document.addEventListener('keydown', aoPressionarTeclaHandler);
  }


  function aoClicarOverlayHandler(evento) {
    if (evento.target.id === 'modalConfiguracoes') fechar();
  }


  function aoPressionarTeclaHandler(evento) {
    if (evento.key !== 'Escape') return;
    const overlay = document.getElementById('modalConfiguracoes');
    if (overlay && !overlay.classList.contains('escondida')) fechar();
  }


  async function aoEnviarFormularioHandler(evento) {
    evento.preventDefault();
    const formulario = evento.target;

    try {
      await salvarPainelAtivo(formulario);
      houveAlteracao = true;
      idEmEdicao = null;
      await renderizarPainelAtivo();
      exibirMensagem('Registro salvo com sucesso.', 'sucesso');
    } catch (erro) {
      exibirMensagem(mensagemDeErro(erro), 'erro');
    }
  }


  function aoMudarAlvoHandler(evento) {
    if (evento.target.name !== 'alvo') return;

    const container = document.getElementById('camposFiltro');
    const formulario = evento.target.form;
    if (!container || !formulario) return;

    // Preserva o que já foi digitado nos campos que continuam visíveis.
    const valores = {
      valor:  valorCampo(formulario, 'valor'),
      evento: valorCampo(formulario, 'evento'),
      host:   valorCampo(formulario, 'host'),
    };
    container.innerHTML = montarCamposFiltro(evento.target.value, valores);
  }


  async function aoClicarConteudoHandler(evento) {
    const botao = evento.target.closest('[data-acao]');
    if (!botao) return;

    const acao = botao.dataset.acao;
    if (acao === 'cancelar') {
      idEmEdicao = null;
      await renderizarPainelAtivo();
      return;
    }

    const id = Number(botao.dataset.id);
    if (acao === 'editar') {
      idEmEdicao = id;
      await renderizarPainelAtivo();
      return;
    }

    if (acao === 'remover') {
      await removerComConfirmacao(id);
    }
  }


  async function removerComConfirmacao(id) {
    if (!window.confirm('Remover este registro? Esta ação não pode ser desfeita.')) return;

    try {
      await removerPainelAtivo(id);
      houveAlteracao = true;
      idEmEdicao = null;
      await renderizarPainelAtivo();
      exibirMensagem('Registro removido.', 'sucesso');
    } catch (erro) {
      exibirMensagem(mensagemDeErro(erro), 'erro');
    }
  }


  return { Inicializar };
})();
