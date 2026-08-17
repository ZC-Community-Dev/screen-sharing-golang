---
description: "Task list for screen share links implementation"
---

# Tasks: Compartilhamento de Tela por Link

**Input**: Design documents from `/specs/001-screen-share-links/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Incluídos. A constituição (princípio III) e o plano exigem TDD:
escrever testes, confirmar falha, depois implementar.

**Organization**: Tasks agrupadas por user story para entrega incremental.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos diferentes, sem dependência)
- **[Story]**: US1–US5 conforme spec.md
- Toda task inclui caminho de arquivo

## Path Conventions

- Backend: `api/` (Go 1.25 + Gin + SQLite)
- Frontend: `app/` (Angular 21 + Tailwind 4 + Vitest)

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Estrutura canônica `api/` + `app/` e dependências

- [x] T001 Create package dirs `api/cmd/server/`, `api/internal/httpapi/`, `api/internal/links/`, `api/internal/ids/`, `api/internal/room/`, `api/internal/db/`, `api/data/`, and `api/testdata/`
- [x] T002 Add Gin, `modernc.org/sqlite`, and a WebSocket library to `api/go.mod`
- [x] T003 [P] Add `app/proxy.conf.json` and wire `ng serve --proxy-config` so `/api` reaches `http://127.0.0.1:8080` in `app/angular.json`
- [x] T004 [P] Ignore `*.db` via `api/data/.gitignore`
- [x] T005 Create Gin process entry skeleton in `api/cmd/server/main.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Infra compartilhada. Nenhuma user story começa antes disto.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 Fail startup when `LINK_ID_SALT` is empty in `api/internal/httpapi/config.go`
- [x] T007 [P] Add structured JSON request logging that never prints salt or tokens in `api/internal/httpapi/log.go`
- [x] T008 [P] Implement `ErrorBody` (`link_not_found`, `link_invalid`, `presenter_unauthorized`, `presenter_conflict`, `share_conflict`, `internal_error`) in `api/internal/httpapi/errors.go`
- [x] T009 Open SQLite at `api/data/links.db`, create the `links` table, and reset `sharing` → `waiting` on open in `api/internal/db/db.go`
- [x] T010 Define `Link` fields and repository interface in `api/internal/links/link.go`
- [x] T011 Register Gin `/api/v1` group and CORS for the Angular origin in `api/internal/httpapi/server.go`
- [x] T012 Wire config, SQLite, logger, and router in `api/cmd/server/main.go`
- [x] T013 Add in-memory session hub (no WebSocket upgrade yet) in `api/internal/room/hub.go`

**Checkpoint**: Foundation ready — user story implementation can begin

---

## Phase 3: User Story 1 - Gerar e copiar um link (Priority: P1) 🎯 MVP

**Goal**: Botão gera id Base62 (≥8) com sal, persiste no SQLite, mostra URL
pública sem token e guarda o token de apresentador só na aba criadora.

**Independent Test**: Gerar dois links, copiar só a URL pública, reiniciar
`api` e `GET /api/v1/links/{id}` continua 200.

### Tests for User Story 1 ⚠️

> Write these tests FIRST and confirm they FAIL before implementation

- [x] T014 [P] [US1] Add failing unit tests for Base62 alphabet, length ≥8, and non-sequential ids in `api/internal/ids/ids_test.go`
- [x] T015 [P] [US1] Add failing contract test for `POST /api/v1/links` 201 `{id,publicUrl,presenterToken}` in `api/internal/httpapi/create_link_contract_test.go`
- [x] T016 [P] [US1] Add failing contract test that `GET /api/v1/links/{id}` 200 omits `presenterToken` in `api/internal/httpapi/get_link_contract_test.go`
- [x] T017 [P] [US1] Add failing integration test create → reopen SQLite file → GET same id in `api/internal/httpapi/link_persist_test.go`
- [x] T018 [P] [US1] Add failing home test: generate copies public URL only in `app/src/app/pages/home/home.spec.ts`

### Implementation for User Story 1

- [x] T019 [US1] Implement HMAC-SHA256(salt)+Base62 (10 chars, retry on collision) in `api/internal/ids/ids.go`
- [x] T020 [US1] Implement `Create` (hash presenter token) and `GetByID` in `api/internal/links/service.go`
- [x] T021 [US1] Implement `POST /links` and `GET /links/:id` (400/404 for bad ids) in `api/internal/httpapi/links.go`
- [x] T022 [US1] Add HTTP client for create/get in `app/src/app/services/links.service.ts`
- [x] T023 [US1] Persist `presenterToken` under `presenterToken:{id}` in `app/src/app/services/presenter-token.store.ts`
- [x] T024 [US1] Build home “Gerar link” + copy public URL in `app/src/app/pages/home/home.ts`
- [x] T025 [US1] Route `''` to Home in `app/src/app/app.routes.ts`

**Checkpoint**: US1 independently testable (generate, copy, persist)

---

## Phase 4: User Story 2 - Apresentador compartilha a tela (Priority: P1)

**Goal**: Quem tem o token entra como apresentador e inicia captura só de
tela. Sem token não apresenta. Sem microfone/câmera.

**Independent Test**: Claim com token inicia share; o mesmo link sem token
é recusado (401); segundo apresentador ativo recebe 409.

### Tests for User Story 2 ⚠️

- [x] T026 [P] [US2] Add failing contract tests for `POST /links/{id}/presenter-sessions` 201/401/409 in `api/internal/httpapi/presenter_session_contract_test.go`
- [x] T027 [P] [US2] Add failing contract tests for `POST /links/{id}/share/start` 200/401/409 in `api/internal/httpapi/share_start_contract_test.go`
- [x] T028 [P] [US2] Add failing hub test: only one active presenter per link in `api/internal/room/hub_presenter_test.go`
- [x] T029 [P] [US2] Add failing room test: share control only with token, no mic/cam in `app/src/app/pages/room/room.spec.ts`

### Implementation for User Story 2

- [x] T030 [US2] Implement `ClaimPresenter` and `StartShare` (hash compare, 409 if presenter exists) in `api/internal/room/hub.go`
- [x] T031 [US2] Implement `POST /links/:id/presenter-sessions` and `POST /links/:id/share/start` in `api/internal/httpapi/sessions.go`
- [x] T032 [US2] Upgrade `GET /links/:id/events?sessionId=` to WebSocket in `api/internal/httpapi/events.go`
- [x] T033 [US2] Add room event client in `app/src/app/services/room-events.service.ts`
- [x] T034 [US2] Capture screen with `getDisplayMedia({ video: true, audio: false })` and create offers in `app/src/app/services/webrtc.service.ts`
- [x] T035 [US2] Presenter room flow (claim + start share) in `app/src/app/pages/room/room.ts`
- [x] T036 [US2] Route `/r/:id` to Room in `app/src/app/app.routes.ts`

**Checkpoint**: US1 + US2 work; presenter can share without voice

---

## Phase 5: User Story 3 - Espectadores assistem pelo link (Priority: P1)

**Goal**: Link público entra como viewer, vê espera ou a tela, sem token.
Id inválido mostra página de erro, sem criar sala.

**Independent Test**: Abrir `/r/{id}` noutro perfil: espera ou tela, sem
botão de apresentar. Id curto/inválido → mensagem de link inválido.

### Tests for User Story 3 ⚠️

- [x] T037 [P] [US3] Add failing contract test for `POST /links/{id}/viewer-sessions` 201 role=viewer in `api/internal/httpapi/viewer_session_contract_test.go`
- [x] T038 [P] [US3] Add failing contract tests for GET malformed (400) and unknown (404) ids in `api/internal/httpapi/get_link_invalid_contract_test.go`
- [x] T039 [P] [US3] Add failing invalid-link page test in `app/src/app/pages/invalid-link/invalid-link.spec.ts`

### Implementation for User Story 3

- [x] T040 [US3] Implement `JoinAsViewer` and presence counts in `api/internal/room/hub.go`
- [x] T041 [US3] Implement `POST /links/:id/viewer-sessions` in `api/internal/httpapi/sessions.go`
- [x] T042 [US3] Relay `signal` / `room.state` / `presence` per `specs/001-screen-share-links/contracts/room-events.md` in `api/internal/room/signal.go`
- [x] T043 [US3] Receive-only peer connections (no display capture) in `app/src/app/services/webrtc.service.ts`
- [x] T044 [US3] Show waiting copy on the stage when `state=waiting` in `app/src/app/components/stage/stage.ts`
- [x] T045 [US3] Build invalid-link page and route `/r/invalid` fallback in `app/src/app/pages/invalid-link/invalid-link.ts`
- [x] T046 [US3] Join `/r/:id` as viewer when `sessionStorage` has no token in `app/src/app/pages/room/room.ts`

**Checkpoint**: Viewers watch or wait; bad ids never create a room

---

## Phase 6: User Story 4 - Sala com visual de reunião (Priority: P2)

**Goal**: Palco central, barra inferior, contagem de pessoas, copiar link.
Sem chat, microfone ou câmera. Visual tipo reunião em tela cheia.

**Independent Test**: Entrar como host e viewer e confirmar palco, barra,
contagem, copiar link, ausência de voz/chat.

### Tests for User Story 4 ⚠️

- [x] T047 [P] [US4] Add failing control-bar test: copy link, people count, no mic/cam/chat in `app/src/app/components/control-bar/control-bar.spec.ts`
- [x] T048 [P] [US4] Add failing stage test: shared video letterboxed, no camera tile in `app/src/app/components/stage/stage.spec.ts`

### Implementation for User Story 4

- [x] T049 [US4] Add Meet-like dark tokens (stage, bar, chrome) in `app/src/styles.css`
- [x] T050 [US4] Layout central stage filling the main area in `app/src/app/components/stage/stage.ts`
- [x] T051 [US4] Persistent bottom control bar in `app/src/app/components/control-bar/control-bar.ts`
- [x] T052 [US4] Bind `participantCount` from `presence` events in `app/src/app/pages/room/room.ts`
- [x] T053 [US4] Copy public `/r/{id}` URL from the bar in `app/src/app/components/control-bar/control-bar.ts`

**Checkpoint**: Room looks like a silent meeting; US1–US3 behavior unchanged

---

## Phase 7: User Story 5 - Encerrar o compartilhamento (Priority: P3)

**Goal**: Stop volta todos à espera; o link e o token continuam válidos;
queda da aba do apresentador também volta à espera; ninguém vira host.

**Independent Test**: Start → stop → start de novo no mesmo token;
fechar aba do host → espera em <5s.

### Tests for User Story 5 ⚠️

- [x] T054 [P] [US5] Add failing contract test for `POST /links/{id}/share/stop` 200 `state=waiting` and GET still 200 in `api/internal/httpapi/share_stop_contract_test.go`
- [x] T055 [P] [US5] Add failing hub test: presenter disconnect broadcasts waiting in `api/internal/room/hub_disconnect_test.go`

### Implementation for User Story 5

- [x] T056 [US5] Implement `StopShare` persisting `waiting` in `api/internal/links/service.go`
- [x] T057 [US5] Implement `POST /links/:id/share/stop` in `api/internal/httpapi/sessions.go`
- [x] T058 [US5] On presenter WebSocket close, set waiting and notify viewers in `api/internal/room/hub.go`
- [x] T059 [US5] Stop and resume share with the same token in `app/src/app/pages/room/room.ts`
- [x] T060 [US5] Return viewers to waiting without reload in `app/src/app/components/stage/stage.ts`

**Checkpoint**: Full session cycle works on one durable link

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Segurança, suíte e validação do quickstart

- [x] T061 [P] Confirm logs and error bodies never include salt, raw token, or token hash in `api/internal/httpapi/log.go`
- [x] T062 [P] Add `LINK_ID_SALT` placeholder (no real secret) in `api/.env.example`
- [x] T063 Run `go test ./...` in `api/` and `npm test` in `app/`
- [x] T064 Walk all scenarios in `specs/001-screen-share-links/quickstart.md`

---

## Phase 9: Analyze remediations (dotenv + token visível)

**Purpose**: Fechar U1/I1/C1 da análise. O sal vive só na API. O Angular
MUST NOT carregar `.env` nem `LINK_ID_SALT`.

**Independent Test**: `go run ./cmd/server` a partir de `api/` sobe lendo
`api/.env` sem exportar variáveis no shell. Gerar link mostra o token na
tela e a URL copiada não o contém.

### Tests ⚠️

> Write these tests FIRST and confirm they FAIL before implementation

- [x] T065 [P] Add failing test that a temp `.env` supplies `LINK_ID_SALT` to `Load` without logging the value in `api/internal/httpapi/dotenv_test.go`
- [x] T066 [P] [US1] Add failing home test: token is visible after generate and absent from the copied URL in `app/src/app/pages/home/home.spec.ts`

### Implementation

- [x] T067 Load `api/.env` with godotenv **before** `httpapi.Load()` in `api/cmd/server/main.go`; OS env wins; do not log salt or tokens
- [x] T068 [US1] Show the presenter token once on the home screen after generate (never in the public URL) in `app/src/app/pages/home/home.ts`
- [x] T069 Document `api/.env` as the local default (shell export still overrides) in `specs/001-screen-share-links/quickstart.md`

**Checkpoint**: `.env` na API funciona; token visível só para quem gerou

---

## Phase 10: Analyze remediations (tela cinza + contador)

**Purpose**: Fechar C1/C2/I1 da análise. Espectador late-join vê a tela;
`participantCount` só conta WebSocket attached e desce no leave.

**Independent Test**: Com partilha ativa, abrir o link noutro perfil mostra
o vídeo (não o palco cinza de espera). Fechar a aba do espectador reduz
o contador nos que ficam.

### Tests ⚠️

> Write these tests FIRST and confirm they FAIL before implementation

- [x] T070 [P] [US3] Document `kind: ready` and `to: presenter` in `specs/001-screen-share-links/contracts/room-events.md`
- [x] T071 [P] [US3] Add failing test: RelaySignal keeps `kind: ready`; `ResolveSignalTo` rewrites `presenter` in `api/internal/room/signal_test.go`
- [x] T075 [P] [US4] Add failing hub test: unattached join does not count; viewer leave decrements attached count in `api/internal/room/hub_presence_test.go`

### Implementation

- [x] T072 [US3] Queue WebSocket frames until `OPEN` in `app/src/app/services/room-events.service.ts`
- [x] T073 [US3] Presenter offers on `ready` without duplicate tracks; ICE queue; STUN in `app/src/app/services/webrtc.service.ts`
- [x] T074 [US3] Viewer sends `ready` on open if already sharing; presenter offers pending viewers on start in `app/src/app/pages/room/room.ts`
- [x] T076 [US4] Count/presence only attached sessions; expire unattached HTTP sessions in `api/internal/room/hub.go`
- [x] T077 [US4] Bind presence without flooring at 1 in `app/src/app/pages/room/room.ts`

**Checkpoint**: Late-join mostra a tela; sair da sala reduz o contador

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: sem dependências
- **Foundational (Phase 2)**: depende da Phase 1 — BLOQUEIA todas as stories
- **US1 (Phase 3)**: depende da Phase 2 — MVP
- **US2 (Phase 4)**: depende da US1 (precisa de `POST/GET /links`)
- **US3 (Phase 5)**: depende da US1 (abrir link) e da US2 (WS/hub) para ver a tela; espera funciona só com join
- **US4 (Phase 6)**: depende da sala US2/US3 (só visual)
- **US5 (Phase 7)**: depende de start share (US2) e viewers (US3)
- **Polish (Phase 8)**: depois das stories desejadas
- **Analyze remediations (Phase 9)**: depois da Phase 8; T067 depende de T065; T068 depende de T066
- **Analyze remediations (Phase 10)**: depois da Phase 9; T072–T074 dependem de T070/T071; T076/T077 dependem de T075

### User Story Dependencies

- **US1 (P1)**: após Phase 2 — nenhuma outra story
- **US2 (P1)**: após US1 API (create/get)
- **US3 (P1)**: após US1 get + hub/WS da US2
- **US4 (P2)**: após a sala existir (US2/US3)
- **US5 (P3)**: após US2 start e US3 viewers

### Within Each User Story

- Testes MUST falhar antes da implementação
- Models/services antes de handlers
- Handlers antes do Angular que os consome
- Story completa antes da próxima prioridade (exceto [P] no mesmo bloco)

### Parallel Opportunities

- T003 e T004 em paralelo após T001
- T007 e T008 em paralelo na Phase 2
- Todos os testes de uma story marcados [P] em paralelo
- US4 testes T047/T048 em paralelo
- T061 e T062 em paralelo no polish
- T065 e T066 em paralelo na Phase 9
- T070, T071 e T075 em paralelo na Phase 10

---

## Parallel Example: User Story 1

```bash
# Testes US1 em paralelo (arquivos diferentes):
Task: "T014 unit tests in api/internal/ids/ids_test.go"
Task: "T015 contract POST in api/internal/httpapi/create_link_contract_test.go"
Task: "T016 contract GET in api/internal/httpapi/get_link_contract_test.go"
Task: "T017 persist test in api/internal/httpapi/link_persist_test.go"
Task: "T018 home spec in app/src/app/pages/home/home.spec.ts"
```

---

## Parallel Example: User Story 2

```bash
Task: "T026 presenter-sessions contract in api/internal/httpapi/presenter_session_contract_test.go"
Task: "T027 share/start contract in api/internal/httpapi/share_start_contract_test.go"
Task: "T028 hub presenter test in api/internal/room/hub_presenter_test.go"
Task: "T029 room spec in app/src/app/pages/room/room.spec.ts"
```

---

## Parallel Example: Phase 9

```bash
Task: "T065 dotenv test in api/internal/httpapi/dotenv_test.go"
Task: "T066 home token visibility in app/src/app/pages/home/home.spec.ts"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Setup
2. Phase 2: Foundational
3. Phase 3: US1
4. **STOP and VALIDATE**: gerar, copiar, persistir após restart
5. Demo do botão “Gerar link”

### Incremental Delivery

1. Setup + Foundational
2. US1 → MVP (links)
3. US2 → apresentar tela
4. US3 → assistir pelo link
5. US4 → visual Meet
6. US5 → encerrar/retomar
7. Polish + quickstart
8. Phase 9: dotenv na API + token visível (FR-004)

### Parallel Team Strategy

1. Time fecha Phase 1+2 junto
2. Depois: um dev em testes US1 (`api/internal/ids` + `httpapi/*_test.go`), outro no Angular home — só após os testes falharem
3. US2/US3 em sequência (mesmo hub/WS)
4. US4 pode começar no `app/src/app/components/` enquanto US5 fecha o backend de stop

---

## Notes

- [P] = arquivos diferentes, sem depender de task incompleta no mesmo arquivo
- Constituição 1.3.0: não criar `backend/` nem `frontend/`
- Token nunca na URL pública; sal nunca em log
- Dotenv só em `api/`; nunca no `app/` Angular
- WebRTC só vídeo; sem SFU
- Commit após cada task ou grupo lógico
- Parar em qualquer checkpoint para validar a story
