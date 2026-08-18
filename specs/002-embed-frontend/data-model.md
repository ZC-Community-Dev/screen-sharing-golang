# Data Model: 002-embed-frontend

Esta feature não cria tabelas nem altera as entidades persistidas da
feature 001. `Link`, `Session` e estado da sala continuam conforme
`../001-screen-share-links/data-model.md`.

## FrontendBundle

Artefato imutável produzido no build e incorporado ao binário Go.
Não é persistido no SQLite.

| Field | Type | Rules |
|---|---|---|
| `root` | embedded filesystem | Subdiretório lógico `dist/browser` |
| `index` | file | `index.html`; obrigatório para startup |
| `assets` | set of files | JS, CSS, imagens, ícones e outros arquivos produzidos pelo Angular |
| `build_time` | build metadata | Opcional; não é necessário em runtime |

**Validation**:

- `index.html` MUST existir e ser arquivo regular.
- Cada caminho servido MUST resolver dentro de `root`.
- O bundle MUST NOT conter `.env`, banco SQLite, salt ou token.
- A cópia MUST remover o bundle anterior antes de gravar o novo.

## StaticRequest

Representa a decisão de roteamento para um pedido não atendido pela API.
Não é persistido.

| Field | Type | Rules |
|---|---|---|
| `method` | enum | Apenas `GET` e `HEAD` são elegíveis para frontend |
| `path` | string | Caminho URL normalizado |
| `reserved` | boolean | Verdadeiro para `/api` e `/api/*` |
| `asset_match` | optional file | Arquivo exato no `FrontendBundle` |

**Resolution states**:

```text
registered API route ------------------------> existing API response
unmatched /api or /api/* --------------------> JSON 404
unmatched non-GET/HEAD ----------------------> 404
GET/HEAD + exact embedded file --------------> static asset
GET/HEAD + no file + non-reserved SPA route -> index.html
startup + index.html absent -----------------> startup error
```

## PublishedService

Unidade de distribuição executável.

| Field | Type | Rules |
|---|---|---|
| `binary` | executable | API Go/Gin |
| `frontend_bundle` | FrontendBundle | Incorporado no executável |
| `runtime_files` | files | `api/.env` e SQLite continuam externos e não públicos |

## Relationships

```text
Angular build 1 --produces--> 1 FrontendBundle
FrontendBundle 1 --embedded in--> 1 PublishedService
PublishedService 1 --serves--> * StaticRequest
```

O bundle é somente leitura em runtime. Atualizar a interface exige novo
build do Angular e novo build do binário.
