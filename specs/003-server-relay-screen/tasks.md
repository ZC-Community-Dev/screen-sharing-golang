---
description: "Task list for adding selectable WebRTC/UDP and WebSocket/WebM server relays"
---

# Tasks: Transporte de mídia WebRTC/UDP ou WebSocket

**Input**: Design documents from `/specs/003-server-relay-screen/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md,
contracts/http-api-v2.yaml, contracts/media-websocket-v2.md,
contracts/frontend-media-config.md, contracts/room-events-v2.md,
quickstart.md

**Existing baseline**: O relay WebRTC/UDP v2 já está implementado e deve
permanecer compatível. Estas tasks cobrem a extensão v2.1 dual-transport.

**Tests**: Obrigatórios. A constituição exige TDD, contratos Gin/WS,
integração Angular→backend, fixtures WebM e prova de fan-out real.

**Organization**: Tasks agrupadas por user story. US1 cria publicação
WebSocket; US2 completa a visualização; US3 prova fan-out; US4 fecha
recovery, privacidade e matriz de navegadores.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: arquivos diferentes e nenhuma dependência incompleta
- **[Story]**: US1–US4 conforme `spec.md`
- Toda task contém caminho exato

## Path Conventions

- Backend/relay: `api/`
- Frontend: `app/`
- Artefatos: `specs/003-server-relay-screen/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Estrutura dual-transport, fixtures e configuração de exemplo.

- [ ] T001 Create dual-transport backend skeletons in `api/internal/media/transport.go`, `api/internal/media/ticket.go`, `api/internal/media/webm.go`, `api/internal/media/buffer.go`, and `api/internal/httpapi/media_websocket.go`
- [ ] T002 [P] Create Angular transport skeletons in `app/src/app/services/media-transport.ts`, `app/src/app/services/webrtc-media.service.ts`, and `app/src/app/services/websocket-media.service.ts`
- [ ] T003 [P] Add non-secret dual-transport defaults (`MEDIA_ALLOWED_TRANSPORTS`, `MEDIA_DEFAULT_TRANSPORT`, `MEDIA_WS_MAX_CHUNK_BYTES`, `MEDIA_WS_MAX_BUFFER_BYTES`) in `api/.env.example`
- [ ] T004 [P] Add deterministic VP8 WebM fixture source and generation notes in `api/internal/media/testdata/README.md` and `api/internal/media/testdata/screen-vp8.webm`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Configuração pública, publicação neutra, tickets e
capabilities usados por todas as stories.

**⚠️ CRITICAL**: nenhuma user story começa antes desta fase.

### Tests ⚠️

> Escrever e executar primeiro; confirmar falha antes da implementação.

- [ ] T005 [P] Add failing tests for allowed/default transport validation, WebSocket limits, and WebSocket-only lazy UDP behavior in `api/internal/httpapi/config_test.go`
- [ ] T006 [P] Add failing `/api/v2/media/config` contract tests proving safe public fields and no UDP IP/port/capacity exposure in `api/internal/httpapi/media_config_contract_test.go`
- [ ] T007 [P] Add failing transport-neutral manager tests for one cross-transport publisher reservation, immutable transport, ownership, and idempotent close in `api/internal/media/transport_test.go`
- [ ] T008 [P] Add failing one-use ticket tests for expiry, hashing, replay, role/link/generation binding, and concurrent consume in `api/internal/media/ticket_test.go`
- [ ] T009 [P] Add failing Angular environment/config tests for non-empty allowed transports, default membership, server intersection, and no automatic fallback in `app/src/app/config.spec.ts`
- [ ] T010 [P] Add failing Angular runtime capability tests for WebRTC and Chrome/Edge WebSocket MIME/MSE matrix in `app/src/app/services/media-transport.spec.ts`

### Implementation

- [ ] T011 Parse and validate dual-transport variables while keeping the Pion engine lazy in `api/internal/httpapi/config.go`
- [ ] T012 Implement safe public media configuration response types and `GET /api/v2/media/config` in `api/internal/httpapi/media_config.go`
- [ ] T013 Register the additive v2.1 media config route without changing existing WebRTC paths in `api/internal/httpapi/server.go`
- [ ] T014 Generalize media room/publication state across `webrtc` and `websocket` with one shared publisher reservation in `api/internal/media/manager.go` and `api/internal/media/transport.go`
- [ ] T015 Implement cryptographically random, hashed, 30-second, single-use WebSocket tickets in `api/internal/media/ticket.go`
- [ ] T016 Add transport/ticket/protocol error mappings without sensitive values in `api/internal/httpapi/errors.go`
- [ ] T017 Add public `allowedMediaTransports` and `defaultMediaTransport` deployment values in `app/src/environments/environment.ts` and `app/src/environments/environment.development.ts`
- [ ] T018 Implement transport types, environment validation, backend intersection, MIME/API checks, and approved Chrome/Edge matrix in `app/src/app/services/media-transport.ts`
- [ ] T019 Extend link/publication/event TypeScript contracts with optional v2.1 fields in `app/src/app/services/links.service.ts` and `app/src/app/services/room-events.service.ts`

**Checkpoint**: configuração cliente/servidor converge; uma sala não pode
reservar publisher WebRTC e WebSocket simultaneamente.

---

## Phase 3: User Story 1 - Apresentador envia a tela ao servidor (Priority: P1) 🎯 MVP técnico

**Goal**: O presenter escolhe WebSocket quando disponível e publica uma
única sequência WebM/VP8 no servidor; WebRTC existente continua intacto.

**Independent Test**: selecionar WebSocket, enviar fixture em mensagens
com cortes arbitrários e confirmar publicação `live`, init e Clusters
VP8 no servidor sem PeerConnection, áudio, transcodificação ou P2P.

### Tests for User Story 1 ⚠️

- [ ] T020 [P] [US1] Add failing WebSocket publisher ticket contract tests for token/role/origin/allowed transport/conflict/capacity in `api/internal/httpapi/media_websocket_publisher_contract_test.go`
- [ ] T021 [P] [US1] Add failing WebSocket upgrade tests for missing, expired, reused, wrong-link tickets and required subprotocol in `api/internal/httpapi/media_websocket_upgrade_test.go`
- [ ] T022 [P] [US1] Add failing incremental WebM parser tests over every split boundary for init, complete Clusters, timestamps, keyframes, VP8-only, and no audio in `api/internal/media/webm_test.go`
- [ ] T023 [P] [US1] Add failing publisher integration test that sends real MediaRecorder-compatible WebM through Gorilla and reaches `connecting→sharing` only after init plus first Cluster in `api/internal/media/websocket_publisher_integration_test.go`
- [ ] T024 [P] [US1] Add failing Angular tests for transport selector defaults, presenter choice, and hiding unsupported WebSocket without fallback in `app/src/app/components/media-transport-selector/media-transport-selector.spec.ts`
- [ ] T025 [P] [US1] Add failing Angular MediaRecorder tests for VP8/video-only, bounded bitrate, 250ms timeslice, binary Blob ordering, bufferedAmount limit, stop, and no PeerConnection in `app/src/app/services/websocket-media.service.spec.ts`

### Implementation for User Story 1

- [ ] T026 [US1] Implement incremental EBML/WebM validation and extraction of init, complete Clusters, media timestamps, and random-access markers in `api/internal/media/webm.go`
- [ ] T027 [US1] Implement time-and-byte bounded immutable Cluster ring with atomic snapshots and zeroization on close in `api/internal/media/buffer.go`
- [ ] T028 [US1] Implement authorized publisher ticket reservation and selected-transport conflict checks in `api/internal/httpapi/media_websocket.go`
- [ ] T029 [US1] Implement same-origin publisher upgrade, one read pump, payload/rate limits, protocol controls, and publication lifecycle in `api/internal/httpapi/media_websocket.go`
- [ ] T030 [US1] Bind WebSocket init/Cluster input to the transport-neutral manager and mark sharing only after valid media in `api/internal/media/manager.go`
- [ ] T031 [US1] Register WebSocket ticket and upgrade routes while preserving existing Pion endpoints in `api/internal/httpapi/server.go`
- [ ] T032 [P] [US1] Implement presenter transport selector UI in `app/src/app/components/media-transport-selector/media-transport-selector.ts` and `app/src/app/components/media-transport-selector/media-transport-selector.html`
- [ ] T033 [US1] Extract the existing Pion publisher/subscriber flow without behavior changes into `app/src/app/services/webrtc-media.service.ts`
- [ ] T034 [US1] Implement WebSocket publisher ticket, MediaRecorder, binary send/backpressure, lifecycle, and cleanup in `app/src/app/services/websocket-media.service.ts`
- [ ] T035 [US1] Coordinate presenter choice before capture and expose connecting/sharing errors in `app/src/app/services/media.service.ts` and `app/src/app/pages/room/room.ts`
- [ ] T036 [US1] Render the selector only for presenter and only before publication in `app/src/app/pages/room/room.html`

**Checkpoint**: presenter envia exatamente uma publicação WebRTC ou
WebSocket ao servidor e nunca alterna automaticamente.

---

## Phase 4: User Story 2 - Espectador recebe a tela do servidor (Priority: P1) 🎯 MVP funcional

**Goal**: Viewer descobre o transporte da publicação e, em WebSocket,
recebe init + buffer reproduzível + Clusters ao vivo por MediaSource.

**Independent Test**: entrar após mais de 10s de publicação WebSocket e
confirmar init-first, no máximo 2s desde random-access, handoff atômico
para live e vídeo visível em <5s sem PeerConnection.

### Tests for User Story 2 ⚠️

- [ ] T037 [P] [US2] Add failing subscriber ticket/upgrade contract tests for no publication, wrong link/session/transport/generation, idempotent close, and capacity in `api/internal/httpapi/media_websocket_subscriber_contract_test.go`
- [ ] T038 [P] [US2] Add failing late-join integration test for init-first, contiguous random-access snapshot ≤2s, no duplicate/gap at live handoff, and reset generation in `api/internal/media/websocket_late_join_integration_test.go`
- [ ] T039 [P] [US2] Add failing room-event tests for optional publication snapshot, `publication.state`, transport IDs, ordering, and old-v2 unknown-event compatibility in `api/internal/httpapi/events_v2_test.go`
- [ ] T040 [P] [US2] Add failing Angular MediaSource tests for serialized append, init/reset, updateend FIFO, live-edge seek, quota cleanup, end, and URL revocation in `app/src/app/services/websocket-media.service.spec.ts`
- [ ] T041 [P] [US2] Add failing room tests for WebSocket late join, transport-directed subscription, waiting teardown, mismatch error, and no viewer selector in `app/src/app/pages/room/room.spec.ts`

### Implementation for User Story 2

- [ ] T042 [US2] Implement viewer ticket reservation bound to current WebSocket publication and generation in `api/internal/httpapi/media_websocket.go`
- [ ] T043 [US2] Implement viewer upgrade, init/bootstrap/live ordering, one write pump, deadlines, and isolated teardown in `api/internal/httpapi/media_websocket.go`
- [ ] T044 [US2] Implement atomic ring snapshot-to-live subscriber registration in `api/internal/media/buffer.go`
- [ ] T045 [US2] Broadcast additive publication descriptors/events without sending binary media over the events socket in `api/internal/room/hub.go` and `api/internal/httpapi/events.go`
- [ ] T046 [US2] Return optional active publication data from v2 link snapshots in `api/internal/httpapi/links.go`
- [ ] T047 [US2] Implement WebSocket viewer ticket, ArrayBuffer receive, MediaSource/SourceBuffer queue, bootstrap, reset, live-edge, and cleanup in `app/src/app/services/websocket-media.service.ts`
- [ ] T048 [US2] Route viewers to the publication transport and reject mismatch without fallback in `app/src/app/services/media.service.ts`
- [ ] T049 [US2] Update room late-join/event lifecycle and remove stale playback on waiting/failed in `app/src/app/pages/room/room.ts`
- [ ] T050 [US2] Support discriminated MediaStream or MediaSource URL playback in `app/src/app/components/stage/stage.ts` and `app/src/app/components/stage/stage.html`

**Checkpoint**: viewer recebe exclusivamente do servidor pelo transporte
selecionado; WebRTC regressions continuam verdes.

---

## Phase 5: User Story 3 - Vários espectadores sem multiplicar o envio (Priority: P2)

**Goal**: 10 viewers WebSocket compartilham Clusters imutáveis; um lento
ou desconectado não bloqueia publisher nem os outros nove.

**Independent Test**: conectar 10 viewers, enviar Clusters, bloquear um,
confirmar fechamento somente dele e continuidade ordenada nos nove,
mantendo um publisher.

### Tests for User Story 3 ⚠️

- [ ] T051 [P] [US3] Add failing 10-viewer WebSocket fan-out test proving one publisher, identical ordered Clusters, and continuation after one close in `api/internal/media/websocket_fanout_integration_test.go`
- [ ] T052 [P] [US3] Add failing slow-consumer test for byte/duration queue limits, close 4429, immutable shared payloads, and unaffected viewers in `api/internal/media/websocket_backpressure_test.go`
- [ ] T053 [P] [US3] Add failing dual-transport capacity tests for shared max rooms/viewers, 11th rejection, and active-session preservation in `api/internal/media/capacity_test.go`

### Implementation for User Story 3

- [ ] T054 [US3] Implement immutable Cluster fan-out with per-viewer bounded queues and safe shared ownership in `api/internal/media/buffer.go`
- [ ] T055 [US3] Enforce synchronized shared limits before WebSocket upgrade/allocation in `api/internal/media/manager.go`
- [ ] T056 [US3] Isolate write deadline, queue overflow, socket failure, and close reason per viewer in `api/internal/httpapi/media_websocket.go`
- [ ] T057 [US3] Expose transport-safe room/subscriber counters without counting media peers as presence in `api/internal/media/manager.go` and `api/internal/room/hub.go`

**Checkpoint**: 10 viewers passam nos dois transportes; consumidor lento
WebSocket é removido sem interromper os demais.

---

## Phase 6: User Story 4 - Recuperação e privacidade (Priority: P2)

**Goal**: Falha/stop/restart apaga buffers/tickets, reconecta somente no
mesmo transporte e mantém mídia/credenciais fora dos logs.

**Independent Test**: interromper publisher WebSocket, observar estados,
retomar com nova geração no mesmo transporte, parar e provar zero bytes,
tickets ou playback restante; Firefox não oferece opção WebSocket.

### Tests for User Story 4 ⚠️

- [ ] T058 [P] [US4] Add failing malformed/truncated/oversized/audio/non-VP8/multi-track and arbitrary-split fuzz tests in `api/internal/media/webm_fuzz_test.go`
- [ ] T059 [P] [US4] Add failing cleanup tests for publisher loss, timeout, reset, stop, manager close, restart semantics, and zero retained buffers/tickets in `api/internal/media/websocket_lifecycle_test.go`
- [ ] T060 [P] [US4] Add failing privacy tests proving token, ticket, query, remote IP, WebM bytes, and close payload never enter logs/errors in `api/internal/httpapi/media_websocket_privacy_test.go`
- [ ] T061 [P] [US4] Add failing heartbeat/origin/rate-limit tests for ping/pong, read timeout, exact production origin, disabled compression, and publisher buffered input in `api/internal/httpapi/media_websocket_upgrade_test.go`
- [ ] T062 [P] [US4] Add failing Angular reconnect/browser tests for same-transport new generation, bounded backoff, cancellation, Chrome/Edge WebSocket visibility, Firefox WebRTC-only, and unsupported-browser error in `app/src/app/services/websocket-media.service.spec.ts` and `app/src/app/services/media-transport.spec.ts`

### Implementation for User Story 4

- [ ] T063 [US4] Implement parser/rate/read limits and categorized protocol failures without payload echo in `api/internal/media/webm.go` and `api/internal/httpapi/media_websocket.go`
- [ ] T064 [US4] Implement ping/pong deadlines, exact origin policy, compression disablement, and terminal socket cleanup in `api/internal/httpapi/media_websocket.go`
- [ ] T065 [US4] Implement generation reset, publisher inactivity transition, subscriber closure, and waiting persistence in `api/internal/media/manager.go` and `api/internal/links/service.go`
- [ ] T066 [US4] Delete consumed/expired tickets and all WebM/parser/queue bytes on every terminal path in `api/internal/media/ticket.go`, `api/internal/media/buffer.go`, and `api/internal/media/manager.go`
- [ ] T067 [US4] Broadcast safe `connecting/live/reconnecting/failed/ended` transitions with transport and generation in `api/internal/httpapi/events.go`
- [ ] T068 [US4] Implement same-WebSocket-transport bounded reconnect with fresh ticket/generation and cancellation on stop/destroy in `app/src/app/services/websocket-media.service.ts`
- [ ] T069 [US4] Render transport-unavailable, reconnecting, failed, and stopped states without stale MediaStream/MediaSource in `app/src/app/pages/room/room.ts` and `app/src/app/components/stage/stage.html`

**Checkpoint**: recovery é visível e sem fallback; nenhum byte/ticket
sobrevive a stop/falha/restart.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Deploy, contratos, regressão WebRTC e validação real.

- [ ] T070 [P] Document proxy upgrade, WSS same-origin, optional UDP, transport config, browser matrix, limits, and capacity in `README.md`
- [ ] T071 [P] Reconcile v2.1 OpenAPI, WebSocket framing, events, and frontend types in `specs/003-server-relay-screen/contracts/` and `app/src/app/services/links.service.ts`
- [ ] T072 Validate real Chrome/Edge MediaRecorder→Go parser→MediaSource playback, random late joins, 10 viewers, and throttled viewer using `specs/003-server-relay-screen/quickstart.md`
- [ ] T073 Run `go test ./...`, WebM fuzz seeds, `go test -race ./internal/media/...` where CGO permits, `npm test`, `npm run build`, and Linux cross-build from `api/` and `app/`
- [ ] T074 Confirm searches contain no participant P2P/fallback, no media on room-events socket, and no SDP/token/ticket/query/WebM logging in `api/` and `app/src/`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup**: sem dependências.
- **Foundational**: depende de Setup e bloqueia todas as stories.
- **US1**: depende de Foundational; cria publisher WebSocket e parser.
- **US2**: depende da publicação/buffer da US1 para playback/late join.
- **US3**: depende de US1 + US2 para fan-out e backpressure.
- **US4**: depende do lifecycle completo US1 + US2; pode iniciar em
  paralelo com US3 após esses checkpoints.
- **Polish**: depende de todas as stories.

### User Story Graph

```text
Setup -> Foundational -> US1 Publisher -> US2 Viewer
                                      ├──> US3 Fan-out/capacity
                                      └──> US4 Recovery/privacy
US3 + US4 -> Polish
```

### Within Each Story

- Testes MUST ser escritos, executados e falhar primeiro.
- Modelo/publicação neutra antes dos sockets.
- Parser e buffer antes de publisher/viewer integration.
- Ticket/HTTP auth antes do upgrade.
- Backend contract antes do cliente Angular.
- Teste real WebM antes de declarar suporte de navegador.
- WebRTC existente permanece verde em cada checkpoint.

### Parallel Opportunities

- T002–T004 executam em paralelo após T001.
- T005–T010 são testes em arquivos distintos.
- T020–T025 podem ser escritos em paralelo antes da US1.
- T037–T041 podem ser escritos em paralelo antes da US2.
- T051–T053 podem ser escritos em paralelo antes da US3.
- T058–T062 podem ser escritos em paralelo antes da US4.
- US3 e US4 podem avançar em paralelo após US2.
- T070 e T071 podem ser executadas em paralelo.

## Parallel Example: User Story 1

```text
T020 publisher ticket/auth contract (Go/Gin)
T021 upgrade/ticket lifecycle contract (Go/Gorilla)
T022 incremental WebM parser (Go)
T023 real WebM publisher integration (Go)
T024 selector/capability UI (Angular)
T025 MediaRecorder publisher (Angular)
```

## Parallel Example: User Story 2

```text
T037 subscriber ticket/upgrade contract
T038 atomic late-join integration
T039 publication room events
T040 MediaSource playback queue
T041 room transport routing
```

## Implementation Strategy

### MVP First

1. Setup + Foundational.
2. US1: presenter publica WebM/VP8 válido no servidor.
3. **STOP AND VALIDATE**: parser, autorização e ausência de P2P.
4. US2: viewer Chrome/Edge reproduz init + buffer + live.
5. **FUNCTIONAL MVP**: WebSocket selecionável ponta a ponta.

### Incremental Delivery

1. Config/manager/tickets neutros, sem alterar WebRTC.
2. Publisher WebSocket.
3. Viewer WebSocket e late join.
4. Dez viewers/backpressure.
5. Recovery/privacy/browser matrix.
6. Deploy/build/quickstart.

### Parallel Team Strategy

Após Foundational:

- Pessoa A: manager, tickets, WebM parser/buffer.
- Pessoa B: contratos Gin/Gorilla e eventos.
- Pessoa C: coordenador Angular, selector, MediaRecorder/MediaSource.

Após US2:

- Pessoa A: fan-out/capacidade.
- Pessoa B: recovery/privacidade.
- Pessoa C: matriz real Chrome/Edge e UX.

## Notes

- WebRTC permanece opção servidor-only e não deve regredir.
- WebSocket de mídia nunca compartilha conexão com room-events.
- Não logar SDP, token, ticket/query, IP remoto ou bytes WebM.
- Não persistir mídia, init, Cluster, ticket ou sessão no SQLite/filesystem.
- O buffer é simultaneamente ≤2s e ≤limite de bytes.
- Firefox oferece WebRTC, não WebSocket, nesta versão.
- Nenhum commit automático.
