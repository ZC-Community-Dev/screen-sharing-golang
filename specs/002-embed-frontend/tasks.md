---
description: "Task list for embedding the Angular frontend in the Go service"
---

# Tasks: Frontend embutido no serviço

**Input**: Design documents from `/specs/002-embed-frontend/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md,
contracts/static-web.md, quickstart.md

**Tests**: Obrigatórios. A constituição (princípios III e IV) exige TDD
e testes de integração na fronteira Gin/Angular.

**Organization**: Tasks agrupadas por user story para permitir entrega
incremental e validação independente.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode executar em paralelo (arquivos diferentes, sem dependência)
- **[Story]**: US1–US4 conforme `spec.md`
- Toda task contém caminho exato

## Path Conventions

- Backend Go/Gin: `api/`
- Frontend Angular: `app/`
- Artefatos da feature: `specs/002-embed-frontend/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Preparar diretórios gerados e regras de versionamento sem
misturar fontes Angular com o backend.

- [x] T001 Create `api/internal/web/dist/.gitkeep`, `api/internal/web/testdata/site/`, and `app/scripts/` according to `specs/002-embed-frontend/plan.md`
- [x] T002 [P] Ignore generated `api/internal/web/dist/browser/` while preserving `api/internal/web/dist/.gitkeep` in `.gitignore`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Fixtures e comando de teste compartilhados pelo handler,
integração Gin e pipeline.

**⚠️ CRITICAL**: nenhuma user story começa antes desta fase.

- [x] T003 [P] Add a minimal SPA fixture (`index.html`, hashed JS, icon) for filesystem tests in `api/internal/web/testdata/site/`
- [x] T004 [P] Add a Node built-in test command `test:build-script` (no new dependency) in `app/package.json`

**Checkpoint**: fixtures e comandos de TDD prontos.

---

## Phase 3: User Story 1 - Abrir um único endereço (Priority: P1) 🎯 MVP

**Goal**: Um handler de frontend validado entrega `/` e assets; o engine
Gin pode recebê-lo sem afetar os testes API-only existentes.

**Independent Test**: subir um engine Gin com o fixture incorporável,
abrir `/` e confirmar HTML 200; pedir um asset e confirmar conteúdo/tipo.

### Tests for User Story 1 ⚠️

> Escrever e executar primeiro; confirmar falha antes da implementação.

- [x] T005 [P] [US1] Add failing unit tests for required `index.html`, `GET /`, `HEAD /`, exact asset serving, and unsupported methods in `api/internal/web/handler_test.go`
- [x] T006 [P] [US1] Add failing Gin integration test proving injected frontend serves `/` through the real engine in `api/internal/httpapi/static_web_test.go`

### Implementation for User Story 1

- [x] T007 [US1] Implement bundle validation and exact static file serving over `fs.FS` in `api/internal/web/handler.go`
- [x] T008 [US1] Add `NewWithFrontend` (keeping `New` for API-only tests) and mount the frontend handler with Gin `NoRoute` in `api/internal/httpapi/server.go`

**Checkpoint**: a raiz e assets funcionam em um único engine Gin usando
fixture; ainda sem depender do build Angular real.

---

## Phase 4: User Story 2 - Convites e salas no mesmo endereço (Priority: P1)

**Goal**: Rotas Angular como `/r/{id}` sobrevivem a abertura direta e
reload; asset ausente não é mascarado por HTML.

**Independent Test**: abrir e recarregar `/r/Abcdefgh12` e `/r/invalid`
no engine de teste; ambos entregam `index.html`, enquanto
`/main-missing.js` retorna 404.

### Tests for User Story 2 ⚠️

> Escrever e executar primeiro; confirmar falha antes da implementação.

- [x] T009 [P] [US2] Add failing unit tests for SPA fallback, missing asset 404, normalized paths, and traversal rejection in `api/internal/web/handler_test.go`
- [x] T010 [P] [US2] Add failing Gin integration tests for direct/reloaded `/r/:id` and `/r/invalid` routes in `api/internal/httpapi/static_web_test.go`

### Implementation for User Story 2

- [x] T011 [US2] Implement safe SPA `index.html` fallback for GET/HEAD routes without file extensions in `api/internal/web/handler.go`

**Checkpoint**: links e reload de sala funcionam no mesmo host.

---

## Phase 5: User Story 3 - Publicar interface junto com o serviço (Priority: P2)

**Goal**: `npm run build` copia uma saída Angular limpa para `api/`;
`go:embed` incorpora o bundle e o processo principal valida-o antes de
escutar.

**Independent Test**: executar build Angular, compilar o binário Go,
copiar apenas o executável e `.env` para outro diretório e abrir `/`.

### Tests for User Story 3 ⚠️

> Escrever e executar primeiro; confirmar falha antes da implementação.

- [x] T012 [P] [US3] Add failing Node tests for missing source, recursive copy, stale destination cleanup, and secret-file rejection in `app/scripts/copy-to-api.test.mjs`
- [x] T013 [P] [US3] Add failing Go tests for embedded bundle subdirectory validation and missing `index.html` error in `api/internal/web/embed_test.go`

### Implementation for User Story 3

- [x] T014 [US3] Implement cross-platform clean/validate/copy from `app/dist/app/browser/` to `api/internal/web/dist/browser/` in `app/scripts/copy-to-api.mjs`
- [x] T015 [US3] Run the copy script after Angular production build and wire Node tests in `app/package.json`
- [x] T016 [US3] Embed `all:dist`, expose validated `dist/browser` as `fs.FS`, and reject missing frontend in `api/internal/web/embed.go`
- [x] T017 [US3] Load the embedded frontend before listening and pass it to `httpapi.NewWithFrontend` in `api/cmd/server/main.go`
- [x] T018 [US3] Execute `npm test`, `npm run test:build-script`, and `npm run build`; verify generated files under `api/internal/web/dist/browser/`

**Checkpoint**: o binário publicado contém a SPA e não depende de
arquivos Angular externos em runtime.

---

## Phase 6: User Story 4 - API continua distinta das telas (Priority: P2)

**Goal**: `/api` permanece reservado; endpoints conhecidos mantêm os
contratos e erros desconhecidos nunca viram `index.html`.

**Independent Test**: com frontend montado, executar endpoint conhecido,
`GET /api/v1/inexistente`, `POST /rota-inexistente` e o WebSocket
existente; somente rotas de frontend recebem HTML.

### Tests for User Story 4 ⚠️

> Escrever e executar primeiro; confirmar falha antes da implementação.

- [x] T019 [US4] Add failing Gin integration tests for known API preservation, JSON 404 under `/api`, non-GET fallback rejection, and WebSocket route precedence in `api/internal/httpapi/static_web_test.go`

### Implementation for User Story 4

- [x] T020 [US4] Reserve `/api` and `/api/*` with JSON 404 and reject non-GET/HEAD requests before frontend fallback in `api/internal/httpapi/server.go`
- [x] T021 [US4] Run existing link/session/WebSocket contract suites with the frontend mounted and fix only integration regressions in `api/internal/httpapi/`

**Checkpoint**: interface e API compartilham origem sem misturar seus
contratos.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Cache, segurança, documentação operacional e validação final.

- [x] T022 [P] Add failing tests for `no-cache` on `index.html`, immutable cache on hashed assets, and no secret filenames in `api/internal/web/handler_test.go`
- [x] T023 Implement cache headers and content types without exposing filesystem paths or secrets in `api/internal/web/handler.go`
- [x] T024 [P] Document production build/run and retained two-process development workflow in `README.md`
- [x] T025 Run `go test ./...`, `npm test`, `npm run test:build-script`, `npm run build`, `go build ./cmd/server`, and all scenarios in `specs/002-embed-frontend/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: sem dependências.
- **Foundational (Phase 2)**: depende da Phase 1; bloqueia todas as stories.
- **US1 (Phase 3)**: depende da Phase 2; cria handler e integração base.
- **US2 (Phase 4)**: depende da US1; expande handler para rotas SPA.
- **US3 (Phase 5)**: depende da US1; usa o handler com bundle real.
- **US4 (Phase 6)**: depende da US1; valida a fronteira API/frontend.
- **Polish (Phase 7)**: depois das stories desejadas; validação final exige todas.

### User Story Dependencies

```text
Setup -> Foundational -> US1
                         ├──> US2
                         ├──> US3
                         └──> US4
US2 + US3 + US4 -> Polish
```

- **US1 (P1)**: MVP técnico; raiz e assets via Gin.
- **US2 (P1)**: requer o handler da US1; independente do bundle real da US3.
- **US3 (P2)**: requer a interface de handler da US1; pode avançar em
  paralelo com US2/US4 após ela.
- **US4 (P2)**: requer montagem Gin da US1; pode avançar em paralelo com US2/US3.

### Within Each User Story

- Tests MUST ser escritos e falhar antes do código de produção.
- Testes de unidade do handler antes da implementação do handler.
- Testes Gin antes de alterar montagem/guards em `server.go`.
- Teste Node antes do script de cópia.
- Script de cópia antes de `go:embed` consumir o bundle real.
- Story completa antes do checkpoint.

### Parallel Opportunities

- T002 pode executar em paralelo com T001.
- T003 e T004 podem executar em paralelo.
- T005 e T006 podem executar em paralelo antes de T007/T008.
- T009 e T010 podem executar em paralelo antes de T011.
- T012 e T013 podem executar em paralelo.
- Depois da US1, US2, preparação de US3 e testes de US4 podem avançar
  em paralelo, respeitando conflitos em `handler_test.go` e
  `static_web_test.go`.
- T022 e T024 podem executar em paralelo.

---

## Parallel Example: User Story 1

```text
Task T005: testes unitários do FS em api/internal/web/handler_test.go
Task T006: testes do engine Gin em api/internal/httpapi/static_web_test.go
```

## Parallel Example: User Story 3

```text
Task T012: testes Node em app/scripts/copy-to-api.test.mjs
Task T013: testes Go de embed em api/internal/web/embed_test.go
```

---

## Implementation Strategy

### MVP First (US1 + US2)

1. Completar Setup e Foundational.
2. Completar US1: Gin entrega raiz/assets.
3. Completar US2: convites/reload funcionam.
4. **STOP AND VALIDATE** com filesystem de fixture.
5. US3 transforma o comportamento validado em binário autocontido.

### Incremental Delivery

1. Setup + Foundational → suporte de teste.
2. US1 → raiz no Gin.
3. US2 → rotas da sala.
4. US3 → pipeline + `go:embed` + binário.
5. US4 → proteção explícita da API.
6. Polish → cache, README e quickstart completo.

### Parallel Team Strategy

Após US1:

- Pessoa A: US2 (`handler.go` / fallback SPA).
- Pessoa B: US3 (`app/scripts`, `package.json`, `embed.go`).
- Pessoa C: US4 (`server.go` / integração API).

Sincronizar antes de editar os testes compartilhados
`handler_test.go` e `static_web_test.go`.

## Notes

- `[P]` significa arquivo diferente e ausência de dependência incompleta.
- Não adicionar dependência Go ou npm para embed/cópia.
- `LINK_ID_SALT`, `.env`, SQLite e tokens nunca entram no bundle.
- O Go serve o Angular compilado; não implementa SSR.
- Não alterar contratos JSON da feature 001.
- Não criar commit automaticamente.
