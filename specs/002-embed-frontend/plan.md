# Implementation Plan: Frontend embutido no serviço

**Branch**: `002-embed-frontend` | **Date**: 2026-08-17 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-embed-frontend/spec.md`

## Summary

O build de produção do Angular será copiado de `app/dist/app/browser/`
para `api/internal/web/dist/` por um script Node multiplataforma. Um
pacote Go dedicado usa `go:embed` para incorporar esse diretório no
binário. O Gin registra primeiro `/api/v1` e, somente em `NoRoute`,
serve arquivos estáticos ou `index.html` como fallback de rotas SPA.
Caminhos desconhecidos sob `/api/` continuam a responder JSON 404.

O Angular permanece um artefato de frontend independente em `app/`;
o backend não renderiza templates nem executa código Angular. Ele
entrega o resultado estático já compilado pelo navegador, permitindo
publicação em um único processo e origem.

## Technical Context

**Language/Version**: Go 1.25.6 em `api/`; TypeScript 5.9 e Angular 21.2 em `app/`

**Primary Dependencies**: Gin 1.12; biblioteca padrão Go `embed`, `io/fs`
e `net/http`; Angular CLI/build 21.2; script Node ESM sem dependência nova

**Storage**: SQLite existente em `api/data/links.db`; esta feature não
adiciona persistência

**Testing**: `go test ./...` com `httptest` para raiz, asset, fallback SPA
e isolamento de `/api/`; Vitest para configuração same-origin; `npm run build`
como teste do pipeline

**Target Platform**: binário Go para Windows ou Linux; navegadores desktop
modernos; Node/npm somente na etapa de build

**Project Type**: aplicação web com frontend SPA e API, publicada como um
único processo

**Performance Goals**: raiz e assets embutidos disponíveis imediatamente
após o processo escutar; tela inicial visível em menos de 15 segundos;
nenhuma leitura de arquivos de frontend do disco em runtime

**Constraints**: manter `api/` e `app/`; `/api/v1` não pode cair no
fallback HTML; sem SSR; sem CDN; sem segredo no bundle; build deve ser
multiplataforma; ausência de `index.html` embutido deve impedir startup

**Scale/Scope**: uma SPA com raiz, `/r/:id`, página inválida e assets
hashados; contratos e regras do compartilhamento existente não mudam

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Evidence |
|---|---|---|
| I. Separação cliente-servidor | PASS | Fontes continuam em `app/` e `api/`; Angular é compilado separadamente e renderizado no navegador. Gin apenas entrega bytes estáticos. |
| II. Contrato HTTP JSON | PASS | Nenhum endpoint de negócio muda. `/api/v1` permanece JSON/WS; documentos HTML e assets não são funcionalidade de negócio. |
| III. Testes primeiro | PASS | Tasks devem criar testes Gin do handler e teste do pipeline antes da implementação. |
| IV. Integração na fronteira | PASS | `httptest` exercitará o engine Gin real e provará que `/api/` nunca retorna a SPA. |
| V. Simplicidade e observabilidade | PASS | Usa apenas biblioteca padrão Go e Node; falha de assets ausentes será explícita no startup, sem registrar segredo. |
| Stack Go/Gin + Angular/Tailwind | PASS | Nenhum framework novo; frontend continua Angular/Tailwind. |
| Árvores `api/` e `app/` | PASS | Fontes não são movidas; somente saída compilada é copiada para `api/internal/web/dist/`. |
| SQLite e Base62 + salt | PASS | Não alterados; salt continua somente em `api/.env`. |

**Post-design re-check**: PASS. O contrato de documentos em
`contracts/static-web.md` reserva `/api/`, define fallback SPA somente
para GET/HEAD e valida `index.html` no startup. O pacote `internal/web`
não importa regras de negócio; `httpapi` apenas conecta seu handler ao
Gin. A separação lógica e os contratos JSON permanecem intactos.

## Project Structure

### Documentation (this feature)

```text
specs/002-embed-frontend/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── static-web.md
└── tasks.md             # gerado depois por /speckit-tasks
```

### Source Code (repository root)

```text
api/
├── cmd/server/main.go
└── internal/
    ├── httpapi/
    │   ├── server.go
    │   └── static_web_test.go
    └── web/
        ├── embed.go
        ├── handler.go
        ├── handler_test.go
        └── dist/
            ├── .gitkeep
            └── browser/       # saída Angular copiada; gerada

app/
├── angular.json
├── package.json
├── scripts/
│   └── copy-to-api.mjs
├── src/
└── dist/app/browser/          # saída local Angular; gerada
```

**Structure Decision**: as duas árvores canônicas permanecem. O script
do `app/` limpa e copia a saída compilada para o pacote Go
`api/internal/web/dist/browser/`. `embed.go` incorpora o diretório;
`handler.go` implementa asset exato, fallback `index.html` e rejeição
de caminhos reservados. `httpapi/server.go` registra esse handler após
as rotas da API.

## Complexity Tracking

Nenhuma violação constitucional. O empacotamento em um binário é uma
etapa de distribuição; não funde fontes, regras de negócio ou
responsabilidades de renderização.
