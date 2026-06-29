// nucleo/bootstrap.js — decide entre /painel/ e /login/ conforme a sessão
// local (usa SignalSessao de sessao.js, carregado antes deste arquivo).

window.location.replace(SignalSessao.Ativa() ? '/painel/' : '/login/');
