# Implementation Plan: Compartilhamento de Tela por Link

**Branch**: `001-screen-share-links` | **Date**: 2026-08-17 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-screen-share-links/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command; its definition describes the execution workflow.

## Summary

Pessoas geram um link público (id Base62 com sal, ≥8 caracteres), guardado
em SQLite, e recebem um token secreto para apresentar só a tela. Quem
abre o link assiste numa sala no estilo reunião, sem voz. O plano de
controle é HTTP JSON no Gin; presença e sinalização WebRTC vão num
WebSocket auxiliar; a mídia é WebRTC só-vídeo em malha.

## Technical Context

**Language/Version**: Go 1.25 (módulo `api`) e TypeScript / Angular 21 (`app`)

**Primary Dependencies**: Gin; Tailwind CSS 4 (já no `app/`);
`modernc.org/sqlite`; WebSocket auxiliar no Gin; WebRTC no navegador
(`getDisplayMedia` sem áudio)

**Storage**: SQLite em `api/data/links.db` (links e hash do token).
Presença e peers WebRTC só em memória.

**Testing**: `go test` + `httptest` (contrato e integração Gin/SQLite);
Vitest no Angular (já configurado)

**Target Platform**: navegadores modernos em desktop (Chrome/Edge/Firefox
com captura de tela); API HTTP local/servidor Windows ou Linux

**Project Type**: aplicação web (API + SPA), árvores `api/` e `app/`

**Performance Goals**: espectador vê a tela em <5s (SC-003); volta à
espera em <5s (SC-008); gerar+copiar link em <30s (SC-001)

**Constraints**: sem contas; sem voz/câmera/chat; um apresentador por
link; token fora da URL; sal e token fora de logs; ids Base62+sal;
SQLite como única fonte de verdade dos links

**Scale/Scope**: ≥10 espectadores simultâneos no mesmo link (malha
WebRTC); sem SFU nesta versão

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Evidence |
|---|---|---|
| I. Separação cliente-servidor | PASS | `api/` (Go/Gin) e `app/` (Angular); UI não no backend |
| II. Contrato HTTP JSON | PASS | [contracts/http-api.yaml](./contracts/http-api.yaml) |
| III. Testes primeiro | PASS | Quickstart e plano exigem `go test` / `npm test` antes de handlers de produção |
| IV. Integração na fronteira | PASS | Contratos + testes httptest no Gin real e fluxo Angular → HTTP |
| V. Simplicidade e observabilidade | PASS | Sem SFU/contas; logs estruturados; sal/token proibidos em log |
| Stack Go+Gin / Angular / Tailwind | PASS | Go+Gin em `api/`; Angular + Tailwind 4 em `app/` |
| Árvores canônicas `api/` e `app/` | PASS | Constituição 1.3.0; sem `backend/` nem `frontend/` |
| SQLite canônico | PASS | `modernc.org/sqlite`, arquivo em `api/data/links.db` |
| IDs Base62 com sal | PASS | HMAC-SHA256 + Base62, `LINK_ID_SALT` |
| Canal além de HTTP JSON | JUSTIFIED | WebSocket (autorizado) + WebRTC (mídia); ver Complexity Tracking |

**Post-design re-check**: os contratos HTTP cobrem criação, abertura,
claim, join, start e stop. WebSocket só transporta presença/estado/SDP.
Nenhum body de espectador inclui token. Layout permanece `api/` +
`app/` (v1.3.0). Gate permanece PASS com a justificativa de mídia
abaixo.

## Project Structure

### Documentation (this feature)

```text
specs/001-screen-share-links/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── http-api.yaml
│   └── room-events.md
└── tasks.md             # Phase 2 (/speckit-tasks) — não criado aqui
```

### Source Code (repository root)

```text
api/
├── cmd/server/          # main Gin
├── internal/
│   ├── httpapi/         # router, handlers, middleware, logs
│   ├── links/           # create/get, token hash, estado
│   ├── ids/             # Base62 + HMAC(sal)
│   ├── room/            # sessões em memória + hub WebSocket
│   └── db/              # open SQLite, schema
├── data/                # links.db (gitignored)
└── testdata/

app/
├── src/app/
│   ├── pages/home/      # botão gerar link
│   ├── pages/room/      # palco + barra (Meet-like, sem voz)
│   ├── pages/invalid-link/
│   ├── components/stage/
│   ├── components/control-bar/
│   └── services/        # links, room events, webrtc (vídeo only)
└── src/styles.css       # @import 'tailwindcss'
```

**Structure Decision**: `api/` (Go/Gin + SQLite) e `app/` (Angular 21 +
Tailwind 4 + Vitest) são as árvores canônicas da constituição 1.3.0.
Renomear, fundir ou inverter essas pastas exige emenda.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Canal WebRTC além de HTTP JSON | Entregar tela ao vivo a vários espectadores (SC-003, SC-005) sem áudio | JPEG/polling no HTTP não sustenta fluidez nem 10 viewers; SFU no servidor é mais complexo |
| WebSocket auxiliar | Presença e start/stop sem reload (FR-011, FR-013, SC-008) | A constituição já autoriza WebSocket; polling HTTP atrasaria a espera |
