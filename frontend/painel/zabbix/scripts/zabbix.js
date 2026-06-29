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
    atencao:  'Atenção',
    info:     'Info',
  };

  const PESO_GRAVIDADE = { desastre: 4, alto: 3, atencao: 2, info: 1 };

  const GRAVIDADE_PADRAO = 'info';
  const ESQUELETO_CARDS  = 6;


  // ----- API pública -----

  async function Renderizar() {
    const container = document.getElementById('cardsZabbix');
    const contador  = document.getElementById('contadorZabbix');
    if (!container || !contador) return;

    container.setAttribute('aria-busy', 'true');
    container.innerHTML = SignalRender.Esqueleto(ESQUELETO_CARDS);
    contador.textContent = '…';

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
    container.setAttribute('aria-busy', 'false');
    const ordenados = ordenarPorGravidade(problemas);
    contador.textContent = `${ordenados.length} ${ordenados.length === 1 ? 'problema' : 'problemas'}`;

    if (ordenados.length === 0) {
      container.innerHTML = SignalRender.BlocoVazio('Nenhum problema pendente.');
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
      horario:   SignalRender.FormatarHorario(p.horario),
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
    const esc = SignalRender.Escapar;
    const rotulo = ROTULO_GRAVIDADE[problema.gravidade] ?? '—';
    const sufixoInstancia = problema.instancia ? ` · ${esc(problema.instancia)}` : '';
    return `
      <article class="card card-zabbix gravidade-${esc(problema.gravidade)}">
        <div class="card-topo">
          <span class="card-titulo">${esc(problema.host)}</span>
          <span class="selo selo-gravidade-${esc(problema.gravidade)}">${esc(rotulo)}</span>
        </div>
        <div class="card-evento">${esc(problema.evento)}</div>
        <div class="card-rodape">
          <span>ID #${esc(String(problema.id))}${sufixoInstancia}</span>
          <span>${esc(problema.horario)}</span>
        </div>
      </article>
    `;
  }


  function mostrarErro(container, contador, erro) {
    container.setAttribute('aria-busy', 'false');
    contador.textContent = '0 problemas';
    container.innerHTML = SignalRender.BlocoErro(`Erro ao carregar problemas Zabbix: ${erro?.message ?? String(erro)}`);
  }


  return { Renderizar };
})();
