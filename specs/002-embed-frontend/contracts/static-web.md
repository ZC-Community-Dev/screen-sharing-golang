# Static Web Contract

Este contrato descreve documentos e arquivos do frontend no mesmo host
da API. Ele complementa, sem alterar, o contrato JSON da feature 001.

## Namespace reservado

- `/api` e `/api/*` são reservados ao backend.
- Rotas registradas em `/api/v1` mantêm método, body, resposta e erros.
- Um caminho reservado desconhecido MUST responder erro JSON 404 e
  MUST NOT retornar `index.html`.
- O upgrade WebSocket existente em `/api/v1/links/{id}/events` continua
  precedendo qualquer fallback do frontend.

## `GET /`

Entrega o documento principal da SPA.

**Success**:

- Status: `200`
- `Content-Type`: `text/html; charset=utf-8`
- Body: `index.html` do bundle incorporado
- `Cache-Control`: `no-cache`

## `GET|HEAD /{asset-path}`

Quando `{asset-path}` corresponde exatamente a arquivo regular no bundle:

- Status: `200`
- `Content-Type`: inferido da extensão/conteúdo
- Body: bytes do arquivo (`HEAD` não inclui body)
- Assets com nome hashado MAY usar
  `Cache-Control: public, max-age=31536000, immutable`

Exemplos: scripts, estilos, favicon e imagens.

Um caminho que parece asset (último segmento contém extensão) mas não
existe SHOULD responder `404`, não `index.html`, para não mascarar build
incompleto.

## `GET|HEAD /{spa-route}`

Quando o caminho:

1. não está sob `/api`;
2. não corresponde a arquivo incorporado; e
3. representa rota do frontend (por exemplo `/r/{id}` ou `/r/invalid`);

o servidor entrega `index.html`:

- Status: `200`
- `Content-Type`: `text/html; charset=utf-8`
- `Cache-Control`: `no-cache`

O Angular resolve a rota e pode consultar a API depois.

## Métodos não elegíveis

`POST`, `PUT`, `PATCH`, `DELETE` e outros métodos em caminhos não
registrados MUST responder `404` ou `405`; MUST NOT devolver a SPA.

## Startup

Antes de escutar:

- abrir o subdiretório incorporado `dist/browser`;
- confirmar `index.html` como arquivo regular;
- em falha, retornar erro que identifique bundle de frontend ausente ou
  inválido;
- não incluir caminhos locais, salt ou tokens no erro.

## Security

- Normalizar e limpar caminhos antes de abrir arquivos.
- Proibir `..` e qualquer escape do root incorporado.
- Nunca incorporar `api/.env`, `api/data/` ou fontes fora do bundle.
- `environment.ts` Angular contém somente configuração pública.
