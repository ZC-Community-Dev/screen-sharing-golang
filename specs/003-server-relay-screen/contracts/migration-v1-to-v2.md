# Migration: room media v1 → v2

Esta alteração é MAJOR porque remove sinalização P2P do WebSocket.

## Endpoint base

- Antigo: `/api/v1`
- Novo: `/api/v2`

Criação/consulta de links, claim presenter, join viewer e stop mantêm os
campos já publicados, agora sob v2. Links/IDs/tokens existentes continuam
válidos.

## Removido

No WebSocket:

- client/server event `signal`;
- `payload.kind`: `ready`, `offer`, `answer`, `ice`;
- campos `to` e `from`;
- alias `to: "presenter"`;
- relé de SDP/ICE entre room session IDs.

No Angular:

- mapa de um PeerConnection por viewer no presenter;
- `createOfferFor(viewerId)`;
- handling de offers/answers/candidates de participantes;
- STUN configurado para conexão browser↔browser.

O servidor v1 MUST NOT continuar retransmitindo sinalização P2P após a
migração. Pode responder erro de versão/410 ou fechar o canal antigo,
mas não manter fallback P2P.

## Adicionado

- `POST /api/v2/links/{id}/media/publisher`;
- `POST /api/v2/links/{id}/media/subscribers`;
- `DELETE /api/v2/links/{id}/media/subscribers/{mediaSessionId}`;
- SDP offer/answer exclusivamente browser↔servidor por HTTP JSON;
- evento `media.state`;
- estados `connecting`, `reconnecting`, `failed`;
- configuração de porta UDP/capacidade do SFU.

## Fluxo presenter v2

1. Claim presenter.
2. Capturar tela, sem áudio.
3. Criar PeerConnection sendonly com o servidor.
4. Esperar ICE gathering completo.
5. POST publisher offer; aplicar server answer.
6. Servidor marca sharing apenas ao receber vídeo.

## Fluxo viewer v2

1. Join viewer + WebSocket v2.
2. Ao receber sharing, criar PeerConnection recvonly com o servidor.
3. Esperar ICE gathering completo.
4. POST subscriber offer; aplicar server answer.
5. Renderizar track recebida do servidor.

## Compatibilidade

- Banco SQLite não requer migração.
- URLs públicas `/r/{id}` não mudam.
- Token armazenado no `sessionStorage` não muda.
- Frontend e backend devem ser publicados juntos; cliente v1 não é
  compatível com mídia v2.

## Extensão dual-transport dentro do v2

A opção WebSocket é a extensão aditiva v2.1 e não exige `/api/v3`:

- endpoints WebRTC publisher/subscriber mantêm método, path e payload;
- `GET /api/v2/media/config` informa transportes públicos;
- `POST /api/v2/links/{id}/media/websocket-tickets` autoriza upgrade;
- `GET /api/v2/links/{id}/media/websocket` transporta WebM binário;
- `publication` em snapshots e `transport`/`publicationId` em eventos
  são campos opcionais;
- `publication.state` é evento novo e ignorável por clientes antigos.

Uma publicação WebSocket não é assistível por frontend v2 antigo.
Backend e frontend dual-transport devem ser implantados juntos antes de
habilitar `websocket`. Não existe fallback para WebRTC ou P2P.
