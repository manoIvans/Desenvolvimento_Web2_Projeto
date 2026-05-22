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

  const TIPO_PADRAO = 'sistema';


  // ----- API publica -----

  async function Renderizar() {
    const container = document.getElementById('cardsMspCloud');
    const contador  = document.getElementById('contadorMspCloud');
    if (!container || !contador) return;

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
    const ordenados = ordenarPorHorarioDecrescente(alertas);
    contador.textContent = `${ordenados.length} ${ordenados.length === 1 ? 'alerta' : 'alertas'}`;

    if (ordenados.length === 0) {
      container.innerHTML = `<div class="vazio">Nenhum alerta detectado.</div>`;
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
      horario:   formatarHorario(a.created_at || a.timestamp || a.horario || ''),
    }));
  }


  function ordenarPorHorarioDecrescente(lista) {
    return [...lista].sort((a, b) => (b.horario || '').localeCompare(a.horario || ''));
  }


  function montarCard(alerta) {
    const rotulo = ROTULO_TIPO[alerta.tipo] ?? '—';
    return `
      <article class="card card-mspcloud tipo-${escapar(alerta.tipo)}">
        <div class="card-topo">
          <span class="card-titulo">${escapar(alerta.cliente)}</span>
          <span class="selo selo-tipo-${escapar(alerta.tipo)}">${escapar(rotulo)}</span>
        </div>
        <div class="card-evento">${escapar(alerta.descricao)}</div>
        <div class="card-rodape">
          <span>ID #${escapar(String(alerta.id))}</span>
          <span>${escapar(alerta.horario)}</span>
        </div>
      </article>
    `;
  }


  function mostrarErro(container, contador, erro) {
    contador.textContent = '0 alertas';
    container.innerHTML = `<div class="vazio">Erro ao carregar alertas MSP: ${escapar(erro?.message ?? String(erro))}</div>`;
  }


  // ----- Utilitarios -----

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


  function formatarHorario(iso) {
    if (!iso) return '—';
    const data = new Date(iso);
    if (isNaN(data.getTime())) return iso;
    const pad = n => String(n).padStart(2, '0');
    return `${data.getFullYear()}-${pad(data.getMonth() + 1)}-${pad(data.getDate())} ${pad(data.getHours())}:${pad(data.getMinutes())}:${pad(data.getSeconds())}`;
  }


  function escapar(texto) {
    return String(texto)
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#39;');
  }


  return { Renderizar };
})();
