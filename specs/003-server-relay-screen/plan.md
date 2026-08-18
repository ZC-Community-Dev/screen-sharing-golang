# Implementation Plan: Transmissão mediada pelo servidor

**Branch**: `003-server-relay-screen` | **Date**: 2026-08-17 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/003-server-relay-screen/spec.md`

## Summary

Substituir a malha WebRTC navegador↔navegador por dois transportes
selecionáveis, ambos terminando no processo Go. No modo `webrtc`, o
apresentador publica uma trilha VP8 para o SFU Pion por UDP e cada viewer
assina a `TrackLocalStaticRTP` do servidor. No modo `websocket`, o
apresentador envia fragmentos binários WebM/VP8 produzidos por
`MediaRecorder`; o servidor valida, mantém somente init + até 2 segundos
em memória e redistribui os bytes aos viewers, que reproduzem por
`MediaSource`. Não há transcodificação, gravação, P2P ou fallback entre
transportes.

O deploy informa transportes permitidos/padrão e o apresentador escolhe
um por publicação. WebRTC mantém negociação HTTP JSON v2 com ICE
não-trickle, ICE Lite e UDP mux. WebSocket usa endpoints de mídia
separados na mesma origem HTTP/HTTPS; presença e estado continuam em um
socket independente. A extensão é aditiva em `/api/v2`: contratos
WebRTC existentes permanecem válidos e novos campos/eventos são
opcionais.

## Technical Context

**Language/Version**: Go 1.25.6 em `api/`; TypeScript 5.9 / Angular 21.2 em `app/`

**Primary Dependencies**: Gin 1.12; Pion WebRTC v4 e Pion ICE v4;
Gorilla WebSocket para presença/estado e sockets binários de mídia
separados; APIs WebRTC, MediaRecorder, MediaSource/SourceBuffer nativas
do navegador; Tailwind 4

**Storage**: SQLite existente somente para links/token/estado. Publicações,
peers, tracks RTP, tickets e buffers WebM ficam em memória e morrem com
o processo.

**Testing**: Go `testing` + `httptest`; integração Pion em processo
publicador→SFU→assinante com RTP sintético; integração WebSocket binária
publisher→buffer→10 viewers; fuzz/unit tests do framing e backpressure;
contratos HTTP/WS v2; Vitest com WebRTC, MediaRecorder e MediaSource
mockados; aceite WebM em Chrome/Edge e regressão WebRTC também em Firefox

**Target Platform**: servidor Windows/Linux; WebRTC em Chrome, Edge e
Firefox desktop; WebSocket/WebM inicialmente em Chrome/Edge validados.
Firefox não anuncia WebSocket enquanto não garantir keyframe VP8 na
janela de 2s. Produção usa TLS reverso HTTP/WSS e UDP quando WebRTC está
habilitado; DTLS-SRTP protege esse transporte.

**Project Type**: aplicação web `api/` + `app/`, distribuída no binário
Go com frontend embutido

**Performance Goals**: tela visível em <5s em 95% das entradas nos dois
transportes; 10 espectadores por sala; saída do apresentador constante
em um fluxo; entrada/saída de viewer não interrompe outros por >1s;
buffer WebSocket limitado a 2 segundos e limite adicional em bytes

**Constraints**: nenhuma conexão browser↔browser; vídeo VP8 sem áudio;
sem transcodificação/gravação; uma publicação e transporte imutável por
link; WebRTC ICE Lite UDP em porta única; WebSocket WebM binário na porta
HTTP/HTTPS; sem fallback automático; `/api/v2`; cliente lento isolado;
SDP, token, ticket, URLs e payload de mídia fora de logs

**Scale/Scope**: mínimo 10 espectadores por link em ambos os transportes;
limites configuráveis por sala e total; um processo nesta versão, sem
cluster, TURN, simulcast, múltiplas qualidades ou transcodificação

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Evidence |
|---|---|---|
| I. Separação cliente-servidor | PASS | Mídia/autorização/capacidade ficam em `api/`; UI e captura ficam em `app/`; contratos públicos separam os lados. |
| II. Contrato HTTP JSON | PASS | Reservas/tickets e SDP usam JSON documentado; upgrades e framing possuem contrato próprio; extensão v2.1 é aditiva e possui nota de migração. |
| III. Testes primeiro | PASS | Plano exige testes Pion, WebM/parser, contratos Gin/WS e Angular antes do código de produção. |
| IV. Integração na fronteira | PASS | Testes provam RTP e WebM binário de publisher→servidor→viewer por handlers reais. |
| V. Simplicidade e observabilidade | PASS | Ambos encaminham bytes comprimidos sem transcodificar; logs excluem SDP, tickets, tokens e mídia. |
| Go/Gin + Angular/Tailwind | PASS | Pion/Gorilla são bibliotecas auxiliares; não substituem Gin nem Angular. |
| SQLite/Base62 + salt | PASS | Link, token e persistência não mudam; mídia não vai ao SQLite. |
| Novo canal UDP/WebRTC | JUSTIFIED | FR-001–FR-004 exigem servidor no caminho da mídia e FR-017 exige a opção UDP; Pion preserva baixa latência e segurança de transporte. |
| WebSocket binário de mídia | JUSTIFIED | FR-017–FR-023 exigem alternativa na mesma porta HTTPS. O upgrade começa em endpoint Gin documentado, usa contrato versionado e fica separado do socket de presença. |

**Post-design re-check**: PASS. O contrato v2 elimina sinalização entre
participantes. WebRTC termina no SFU; WebSocket termina em handlers Gin
documentados e nunca compartilha conexão com presença. O modelo torna
mídia efêmera e isolada por link; o buffer WebSocket possui limites de
tempo/bytes e limpeza terminal. Pion/Gorilla adicionam canais
justificados sem novo serviço ou framework HTTP.

## Project Structure

### Documentation (this feature)

```text
specs/003-server-relay-screen/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── http-api-v2.yaml
│   ├── frontend-media-config.md
│   ├── media-websocket-v2.md
│   ├── room-events-v2.md
│   └── migration-v1-to-v2.md
└── tasks.md             # gerado por /speckit-tasks
```

### Source Code (repository root)

```text
api/
├── cmd/server/main.go
└── internal/
    ├── httpapi/
    │   ├── media.go
    │   ├── media_contract_test.go
    │   ├── server.go
    │   └── config.go
    ├── media/
    │   ├── engine.go        # Pion API, ICE Lite, UDP mux
    │   ├── manager.go       # publicação neutra, transporte e capacidade
    │   ├── publisher.go     # oferta/answer e TrackRemote
    │   ├── subscriber.go    # oferta/answer e TrackLocal
    │   ├── relay.go         # RTP forwarding + RTCP/PLI
    │   ├── websocket.go     # upgrade, auth e lifecycle WebM
    │   ├── webm.go          # framing/init e validação de fragmentos
    │   ├── buffer.go        # ring efêmero por tempo e bytes
    │   └── *_test.go
    ├── room/
    │   ├── hub.go           # presença/estado, sem SendTo
    │   └── signal.go        # removido
    └── web/                 # frontend embed existente

app/
└── src/app/
    ├── pages/room/room.ts
    ├── services/
    │   ├── media.service.ts       # coordenador de transporte
    │   ├── webrtc-media.service.ts
    │   ├── websocket-media.service.ts
    │   ├── media.service.spec.ts
    │   ├── links.service.ts       # HTTP v2
    │   └── room-events.service.ts # state/presence apenas
    ├── components/media-transport-selector/
    └── components/stage/          # MediaStream ou MediaSource URL
```

**Structure Decision**: `api/internal/media` mantém publicação,
capacidade e lifecycle neutros; implementações Pion e WebSocket ficam
separadas sob o mesmo manager. `httpapi/media.go` autoriza SDP JSON e
emite tickets WebSocket curtos/uso único antes do upgrade, sem token em
URL. O Angular usa um coordenador e serviços específicos: um
PeerConnection servidor no modo WebRTC ou um MediaRecorder/MediaSource
no modo WebSocket. Eventos da sala apenas anunciam estado, publicação e
transporte.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Canal UDP/DTLS-SRTP via Pion | FR-017 exige opção UDP e Firefox permanece suportado por WebRTC | Remover WebRTC eliminaria o transporte de menor latência e deixaria navegadores sem keyframes WebM frequentes sem opção suportada. |
| API `/api/v2` | Remover o evento `signal` e o handshake P2P é incompatível | Alterar `/api/v1` silenciosamente viola o princípio II e quebra clientes antigos sem migração explícita. |
| Segundo transporte WebSocket/WebM | FR-017–FR-023 exigem alternativa explícita ao UDP na mesma porta HTTPS | Fallback automático contraria a escolha do usuário; WebTransport/codec customizado amplia incompatibilidade e complexidade. |
| Buffer efêmero de inicialização | Viewer tardio precisa iniciar em <5s no modo WebSocket | Sem buffer, precisa aguardar segmento/keyframe futuro; histórico completo viola privacidade e memória limitada. |
