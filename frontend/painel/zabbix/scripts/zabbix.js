// ═══════════════════════════════════════════════
//  zabbix/scripts/zabbix.js — Renderiza cards de problemas Zabbix
//  consumindo o endpoint GET /zabbix do backend.
// ═══════════════════════════════════════════════

const ZabbixSecao = (function () {

  // ----- Constantes -----

  const MAPA_PRIO_PARA_GRAVIDADE = {
    'Disaster':       'desastre',
    'High':           'alto',
    'Average':        'atencao',
    'Warning':        'atencao',
    'Information':    'info',
    'Not classified': 'info',
  };

  const ROTULO_GRAVIDADE = {
    desastre: 'Desastre',
    alto:     'Alto',
    atencao:  'Atencao',
    info:     'Info',
  };

  const PESO_GRAVIDADE = { desastre: 4, alto: 3, atencao: 2, info: 1 };

  const GRAVIDADE_PADRAO = 'info';


  // ----- API publica -----

  async function Renderizar() {
    const container = document.getElementById('cardsZabbix');
    const contador  = document.getElementById('contadorZabbix');
    if (!container || !contador) return;

    let problemas;
    try {
      const resposta = await SignalApi.BuscarZabbix();
      problemas = converterProblemas(resposta?.data ?? []);
    } catch (erro) {
      mostrarErro(container, contador, erro);
      return;
    }

    desenharCards(container, contador, problemas);
  }


  // ----- Internos -----

  function desenharCards(container, contador, problemas) {
    const ordenados = ordenarPorGravidade(problemas);
    contador.textContent = `${ordenados.length} ${ordenados.length === 1 ? 'problema' : 'problemas'}`;

    if (ordenados.length === 0) {
      container.innerHTML = `<div class="vazio">Nenhum problema pendente.</div>`;
      return;
    }

    container.innerHTML = ordenados.map(montarCard).join('');
  }


  function converterProblemas(lista) {
    return lista.map((p, indice) => ({
      id:        indice + 1,
      host:      p.host || p.instancia || '—',
      evento:    p.evento || p.mensagem || '—',
      gravidade: MAPA_PRIO_PARA_GRAVIDADE[p.prio_label] ?? GRAVIDADE_PADRAO,
      horario:   formatarHorario(p.horario),
      instancia: p.instancia || '',
    }));
  }


  function ordenarPorGravidade(lista) {
    return [...lista].sort((a, b) => {
      const pa = PESO_GRAVIDADE[a.gravidade] ?? 0;
      const pb = PESO_GRAVIDADE[b.gravidade] ?? 0;
      if (pb !== pa) return pb - pa;
      return (b.horario || '').localeCompare(a.horario || '');
    });
  }


  function montarCard(problema) {
    const rotulo = ROTULO_GRAVIDADE[problema.gravidade] ?? '—';
    const sufixoInstancia = problema.instancia ? ` · ${escapar(problema.instancia)}` : '';
    return `
      <article class="card card-zabbix gravidade-${escapar(problema.gravidade)}">
        <div class="card-topo">
          <span class="card-titulo">${escapar(problema.host)}</span>
          <span class="selo selo-gravidade-${escapar(problema.gravidade)}">${escapar(rotulo)}</span>
        </div>
        <div class="card-evento">${escapar(problema.evento)}</div>
        <div class="card-rodape">
          <span>ID #${escapar(String(problema.id))}${sufixoInstancia}</span>
          <span>${escapar(problema.horario)}</span>
        </div>
      </article>
    `;
  }


  function mostrarErro(container, contador, erro) {
    contador.textContent = '0 problemas';
    container.innerHTML = `<div class="vazio">Erro ao carregar problemas Zabbix: ${escapar(erro?.message ?? String(erro))}</div>`;
  }


  // ----- Utilitarios -----

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
