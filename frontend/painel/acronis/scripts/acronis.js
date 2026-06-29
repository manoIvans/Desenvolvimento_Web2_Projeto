// ═══════════════════════════════════════════════
//  acronis/scripts/acronis.js — Renderiza cards de alertas Acronis Cyber
//  Protect Cloud consumindo o endpoint GET /acronis do backend.
// ═══════════════════════════════════════════════

const AcronisSecao = (function () {

  // ----- Constantes -----

  // Severidade da API Acronis → classe visual interna.
  const MAPA_SEVERIDADE = {
    critical:    'critico',
    error:       'erro',
    warning:     'aviso',
    information: 'info',
    ok:          'ok',
  };

  const ROTULO_SEVERIDADE = {
    critico: 'Crítico',
    erro:    'Erro',
    aviso:   'Aviso',
    info:    'Info',
    ok:      'OK',
  };

  const PESO_SEVERIDADE = { critico: 5, erro: 4, aviso: 3, info: 2, ok: 1 };

  const SEVERIDADE_PADRAO = 'info';
  const ESQUELETO_CARDS   = 6;


  // ----- API pública -----

  async function Renderizar() {
    const container = document.getElementById('cardsAcronis');
    const contador  = document.getElementById('contadorAcronis');
    if (!container || !contador) return;

    container.setAttribute('aria-busy', 'true');
    container.innerHTML = SignalRender.Esqueleto(ESQUELETO_CARDS);
    contador.textContent = '…';

    let alertas;
    try {
      const resposta = await SignalApi.BuscarAcronis();
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
    const ordenados = ordenarPorSeveridade(alertas);
    contador.textContent = `${ordenados.length} ${ordenados.length === 1 ? 'alerta' : 'alertas'}`;

    if (ordenados.length === 0) {
      container.innerHTML = SignalRender.BlocoVazio('Nenhum alerta Acronis.');
      return;
    }

    container.innerHTML = ordenados.map(montarCard).join('');
  }


  function converterAlertas(lista) {
    return lista.map(a => ({
      titulo:     a.titulo || a.tenant || '—',
      descricao:  a.descricao || a.categoria || '—',
      severidade: MAPA_SEVERIDADE[String(a.severidade || '').toLowerCase()] ?? SEVERIDADE_PADRAO,
      tenant:     a.tenant || '',
      horario:    SignalRender.FormatarHorario(a.horario),
    }));
  }


  function ordenarPorSeveridade(lista) {
    return [...lista].sort((a, b) => {
      const pa = PESO_SEVERIDADE[a.severidade] ?? 0;
      const pb = PESO_SEVERIDADE[b.severidade] ?? 0;
      if (pb !== pa) return pb - pa;
      return (b.horario || '').localeCompare(a.horario || '');
    });
  }


  function montarCard(alerta) {
    const esc = SignalRender.Escapar;
    const rotulo = ROTULO_SEVERIDADE[alerta.severidade] ?? '—';
    const sufixoTenant = alerta.tenant ? `tenant ${esc(alerta.tenant)}` : '';
    return `
      <article class="card card-acronis sev-${esc(alerta.severidade)}">
        <div class="card-topo">
          <span class="card-titulo">${esc(alerta.titulo)}</span>
          <span class="selo selo-acronis-${esc(alerta.severidade)}">${esc(rotulo)}</span>
        </div>
        <div class="card-evento">${esc(alerta.descricao)}</div>
        <div class="card-rodape">
          <span>${sufixoTenant}</span>
          <span>${esc(alerta.horario)}</span>
        </div>
      </article>
    `;
  }


  function mostrarErro(container, contador, erro) {
    container.setAttribute('aria-busy', 'false');
    contador.textContent = '0 alertas';
    container.innerHTML = SignalRender.BlocoErro(`Erro ao carregar alertas Acronis: ${erro?.message ?? String(erro)}`);
  }


  return { Renderizar };
})();
