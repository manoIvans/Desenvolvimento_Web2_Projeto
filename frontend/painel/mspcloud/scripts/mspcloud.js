// ═══════════════════════════════════════════════
//  mspcloud/scripts/mspcloud.js — Renderiza cards de alertas MSP Clouds
//  consumindo o endpoint GET /mspclouds do backend.
// ═══════════════════════════════════════════════

const MspCloudSecao = (function () {

  // ----- Constantes -----

  const MAPA_KEYWORD_PARA_TIPO = {
    cloudbackup:        'backup',
    msp_n_central:      'rede',
    msp_cyber_security: 'sistema',
    msp_n_able:         'sistema',
  };

  const ROTULO_TIPO = {
    backup:  'Backup',
    rede:    'Rede',
    sistema: 'Sistema',
  };

  const TIPO_PADRAO     = 'sistema';
  const ESQUELETO_CARDS = 6;


  // ----- API pública -----

  async function Renderizar() {
    const container = document.getElementById('cardsMspCloud');
    const contador  = document.getElementById('contadorMspCloud');
    if (!container || !contador) return;

    container.setAttribute('aria-busy', 'true');
    container.innerHTML = SignalRender.Esqueleto(ESQUELETO_CARDS);
    contador.textContent = '…';

    let alertas;
    try {
      const resposta = await SignalApi.BuscarMsp();
      alertas = converterAlertas(resposta?.data ?? []);
    } catch (erro) {
      mostrarErro(container, contador, erro);
      return;
    }

    desenharCards(container, contador, alertas);
  }


  // ----- Internos -----

  function desenharCards(container, contador, alertas) {
    container.setAttribute('aria-busy', 'false');
    const ordenados = ordenarPorHorarioDecrescente(alertas);
    contador.textContent = `${ordenados.length} ${ordenados.length === 1 ? 'alerta' : 'alertas'}`;

    if (ordenados.length === 0) {
      container.innerHTML = SignalRender.BlocoVazio('Nenhum alerta detectado.');
      return;
    }

    container.innerHTML = ordenados.map(montarCard).join('');
  }


  function converterAlertas(lista) {
    return lista.map((a, indice) => ({
      id:        indice + 1,
      cliente:   a.client || a.cliente || '—',
      tipo:      MAPA_KEYWORD_PARA_TIPO[a.product_keyword] ?? TIPO_PADRAO,
      descricao: extrairDescricao(a),
      horario:   SignalRender.FormatarHorario(a.created_at || a.timestamp || a.horario || ''),
    }));
  }


  function ordenarPorHorarioDecrescente(lista) {
    return [...lista].sort((a, b) => (b.horario || '').localeCompare(a.horario || ''));
  }


  function montarCard(alerta) {
    const esc = SignalRender.Escapar;
    const rotulo = ROTULO_TIPO[alerta.tipo] ?? '—';
    return `
      <article class="card card-mspcloud tipo-${esc(alerta.tipo)}">
        <div class="card-topo">
          <span class="card-titulo">${esc(alerta.cliente)}</span>
          <span class="selo selo-tipo-${esc(alerta.tipo)}">${esc(rotulo)}</span>
        </div>
        <div class="card-evento">${esc(alerta.descricao)}</div>
        <div class="card-rodape">
          <span>ID #${esc(String(alerta.id))}</span>
          <span>${esc(alerta.horario)}</span>
        </div>
      </article>
    `;
  }


  function mostrarErro(container, contador, erro) {
    container.setAttribute('aria-busy', 'false');
    contador.textContent = '0 alertas';
    container.innerHTML = SignalRender.BlocoErro(`Erro ao carregar alertas MSP: ${erro?.message ?? String(erro)}`);
  }


  // ----- Utilitários -----

  function extrairDescricao(alerta) {
    if (typeof alerta.message === 'string') return alerta.message;
    if (alerta.message && typeof alerta.message === 'object') {
      return alerta.message.error
          || alerta.message.description
          || alerta.message.subject
          || resumirObjeto(alerta.message);
    }
    return alerta.type || alerta.descricao || '—';
  }


  function resumirObjeto(objeto) {
    const partes = [];
    for (const [chave, valor] of Object.entries(objeto)) {
      if (valor === null || valor === undefined || valor === '') continue;
      partes.push(`${chave}: ${valor}`);
      if (partes.length >= 3) break;
    }
    return partes.join(' · ') || '—';
  }


  return { Renderizar };
})();
