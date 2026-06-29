// ═══════════════════════════════════════════════
//  configuracoes/scripts/configuracoes.js
//  Modal de configurações — CRUD de instâncias Zabbix, instâncias MSP
//  Clouds e contas Acronis, persistidos no backend. Os filtros (exclusivos
//  do Zabbix) têm janela própria (filtros/scripts/filtros.js).
// ═══════════════════════════════════════════════

const ConfiguracoesModal = (function () {

  // ----- Constantes -----

  const PAINEL_PADRAO = 'zabbix';

  const EVENTO_DADOS_ALTERADOS = 'configuracoesAlteradasEvent';

  const PAINEIS = {
    zabbix:   () => renderizarPainelZabbix(),
    mspcloud: () => renderizarPainelMsp(),
    acronis:  () => renderizarPainelAcronis(),
  };


  // ----- Estado -----

  let painelAtivo = PAINEL_PADRAO;
  let idEmEdicao  = null;
  let houveAlteracao = false;

  const escapar = SignalRender.Escapar;

  const modal = SignalModal.Criar({
    overlayId: 'modalConfiguracoes',
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
    houveAlteracao = false;
    modal.abrir(document.getElementById('botaoFecharConfiguracoes'));
    trocarPainel(PAINEL_PADRAO);
  }


  function trocarPainel(idPainel) {
    painelAtivo = idPainel;
    idEmEdicao  = null;
    document.querySelectorAll('.modal-aba').forEach(aba => {
      const ativa = aba.dataset.painel === idPainel;
      aba.classList.toggle('ativa', ativa);
      aba.setAttribute('aria-selected', ativa ? 'true' : 'false');
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
      conteudo.innerHTML = `<div class="config-mensagem erro">${escapar(SignalModal.MensagemDeErro(erro))}</div>`;
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
        ${SignalModal.BotoesForm(idEmEdicao)}
      </form>
      <div class="config-mensagem" data-mensagem role="alert" aria-live="polite"></div>
      <ul class="config-lista">
        ${lista.map(montarLinhaZabbix).join('') || SignalModal.ItemVazio('Nenhuma instância Zabbix cadastrada.')}
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
        ${SignalModal.BotoesItem(instancia.id)}
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
        ${SignalModal.BotoesForm(idEmEdicao)}
      </form>
      <div class="config-mensagem" data-mensagem role="alert" aria-live="polite"></div>
      <ul class="config-lista">
        ${lista.map(montarLinhaMsp).join('') || SignalModal.ItemVazio('Nenhuma instância MSP Clouds cadastrada.')}
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
        ${SignalModal.BotoesItem(instancia.id)}
      </li>
    `;
  }


  // ----- Painel: contas Acronis -----

  async function renderizarPainelAcronis() {
    const resposta = await SignalApi.ListarContasAcronis();
    const lista = resposta?.data ?? [];
    const emEdicao = lista.find(c => c.id === idEmEdicao);

    document.getElementById('modalConteudo').innerHTML = `
      <form class="config-form" data-form="acronis">
        <h3>${idEmEdicao ? 'Editar conta Acronis' : 'Nova conta Acronis'}</h3>
        <label>Nome
          <input name="nome" type="text" maxlength="120"
                 value="${escapar(emEdicao?.nome ?? '')}" placeholder="Tenant principal" />
        </label>
        <p class="config-dica">Informe a URL do datacenter <em>ou</em> o login (a URL é descoberta a partir dele).</p>
        <label>URL do datacenter
          <input name="server_url" type="url" maxlength="500"
                 value="${escapar(emEdicao?.server_url ?? '')}"
                 placeholder="https://eu2-cloud.acronis.com" />
        </label>
        <label>Login
          <input name="login" type="text" maxlength="200"
                 value="${escapar(emEdicao?.login ?? '')}" placeholder="operador@empresa" />
        </label>
        <label>Client ID *
          <input name="client_id" type="text" maxlength="200" required
                 value="${escapar(emEdicao?.client_id ?? '')}" placeholder="id do cliente da API" />
        </label>
        <label>Client Secret *
          <input name="client_secret" type="text" maxlength="500" required
                 value="${escapar(emEdicao?.client_secret ?? '')}" placeholder="segredo do cliente da API" />
        </label>
        ${SignalModal.BotoesForm(idEmEdicao)}
      </form>
      <div class="config-mensagem" data-mensagem role="alert" aria-live="polite"></div>
      <ul class="config-lista">
        ${lista.map(montarLinhaAcronis).join('') || SignalModal.ItemVazio('Nenhuma conta Acronis cadastrada.')}
      </ul>
    `;
  }


  function montarLinhaAcronis(conta) {
    const destino = conta.server_url
      ? conta.server_url
      : (conta.login ? `datacenter via login ${conta.login}` : '—');
    return `
      <li class="config-item">
        <div class="config-item-info">
          <span class="config-item-titulo">${escapar(conta.nome || conta.client_id || 'conta sem nome')}</span>
          <span class="config-item-detalhe">${escapar(destino)}</span>
          <span class="config-item-detalhe">client id: ${escapar(conta.client_id)}</span>
          <span class="config-item-detalhe">secret: ${escapar(mascarar(conta.client_secret))}</span>
        </div>
        ${SignalModal.BotoesItem(conta.id)}
      </li>
    `;
  }


  // ----- Ações: salvar -----

  async function salvarPainelAtivo(formulario) {
    if (painelAtivo === 'zabbix')   return await salvarZabbix(formulario);
    if (painelAtivo === 'mspcloud') return await salvarMsp(formulario);
    return await salvarAcronis(formulario);
  }


  async function salvarZabbix(formulario) {
    const dados = {
      nome:    SignalModal.ValorCampo(formulario, 'nome'),
      url:     SignalModal.ValorCampo(formulario, 'url'),
      api_key: SignalModal.ValorCampo(formulario, 'api_key'),
    };
    if (idEmEdicao) {
      await SignalApi.AtualizarInstanciaZabbix(idEmEdicao, dados);
    } else {
      await SignalApi.CriarInstanciaZabbix(dados);
    }
  }


  async function salvarMsp(formulario) {
    const dados = { api_key: SignalModal.ValorCampo(formulario, 'api_key') };
    if (idEmEdicao) {
      await SignalApi.AtualizarInstanciaMsp(idEmEdicao, dados);
    } else {
      await SignalApi.CriarInstanciaMsp(dados);
    }
  }


  async function salvarAcronis(formulario) {
    const dados = {
      nome:          SignalModal.ValorCampo(formulario, 'nome'),
      server_url:    SignalModal.ValorCampo(formulario, 'server_url'),
      login:         SignalModal.ValorCampo(formulario, 'login'),
      client_id:     SignalModal.ValorCampo(formulario, 'client_id'),
      client_secret: SignalModal.ValorCampo(formulario, 'client_secret'),
    };
    if (idEmEdicao) {
      await SignalApi.AtualizarContaAcronis(idEmEdicao, dados);
    } else {
      await SignalApi.CriarContaAcronis(dados);
    }
  }


  // ----- Ações: remover -----

  async function removerPainelAtivo(id) {
    if (painelAtivo === 'zabbix')   return await SignalApi.RemoverInstanciaZabbix(id);
    if (painelAtivo === 'mspcloud') return await SignalApi.RemoverInstanciaMsp(id);
    return await SignalApi.RemoverContaAcronis(id);
  }


  // ----- Utilitários gerais -----

  function exibirMensagem(texto, tipo) {
    SignalModal.ExibirMensagem(document.getElementById('modalConteudo'), texto, tipo);
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


  // ----- Handlers / listeners (sempre no final) -----

  function registrarHandlers() {
    const botaoAbrir = document.getElementById('botaoConfiguracoes');
    if (botaoAbrir) botaoAbrir.addEventListener('click', abrir);

    const botaoFechar = document.getElementById('botaoFecharConfiguracoes');
    if (botaoFechar) botaoFechar.addEventListener('click', () => modal.fechar());

    document.querySelectorAll('.modal-aba').forEach(aba => {
      aba.addEventListener('click', () => trocarPainel(aba.dataset.painel));
    });

    const conteudo = document.getElementById('modalConteudo');
    if (conteudo) {
      conteudo.addEventListener('submit', aoEnviarFormularioHandler);
      conteudo.addEventListener('click', aoClicarConteudoHandler);
    }
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
      exibirMensagem(SignalModal.MensagemDeErro(erro), 'erro');
    }
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
      exibirMensagem(SignalModal.MensagemDeErro(erro), 'erro');
    }
  }


  return { Inicializar };
})();
