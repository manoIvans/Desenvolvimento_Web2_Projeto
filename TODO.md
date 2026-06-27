# TODO — Pendências do Projeto

- [ ] CSP usa `'unsafe-inline'` em internal/seguranca/cabecalhos.go:18 — o frontend ainda tem script/estilo inline (frontend/index.html); mover esses inlines para arquivos externos permite endurecer a CSP removendo `'unsafe-inline'`.
- [ ] Mapa de baldes do rate limiter cresce sem limpeza em internal/seguranca/limitador.go:33 — não há GC dos IPs ociosos; adicionar uma varredura periódica que remova baldes recarregados e inativos.
