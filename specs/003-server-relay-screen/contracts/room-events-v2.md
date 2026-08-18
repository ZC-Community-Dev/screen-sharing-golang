# Room Events v2 (WebSocket)

**Path**: `GET /api/v2/links/{id}/events?sessionId={sessionId}`

O upgrade exige sessão HTTP válida e conectada ao mesmo link. Frames são
JSON texto. Este canal transporta somente presença/estado; SDP, ICE, RTP
e fragmentos WebM não passam aqui. Mídia WebSocket usa conexão separada.

## Client → Server

Nenhum frame de sinalização é aceito.

- `signal`, `ready`, `offer`, `answer`, `ice`, `to` e `from` são
  inválidos no v2.
- Frame recebido do cliente MUST ser ignorado ou provocar fechamento com
  policy violation; MUST NOT ser retransmitido.
- Keepalive WebSocket pode usar ping/pong de protocolo, não frame JSON.

## Server → Client

### `room.state`

```json
{
  "type": "room.state",
  "payload": {
    "state": "sharing",
    "publication": {
      "id": "opaque-id",
      "transport": "websocket",
      "state": "live"
    }
  }
}
```

`state`: `waiting` | `connecting` | `sharing` | `reconnecting` | `failed`.

- `connecting`: publisher negociado, aguardando vídeo.
- `sharing`: servidor recebeu track e pode aceitar subscribers.
- `reconnecting`: publisher perdeu conectividade temporariamente.
- `failed`: mídia falhou; UI informa antes de voltar a waiting.

Enviado ao conectar e a cada transição.

`publication` é opcional/`null` em `waiting`, permitindo que late joiners
descubram o transporte sem depender de evento anterior.

### `publication.state`

```json
{
  "type": "publication.state",
  "payload": {
    "publicationId": "opaque-id",
    "transport": "websocket",
    "state": "live"
  }
}
```

`transport`: `webrtc` | `websocket`.

`state`: `connecting` | `live` | `reconnecting` | `failed` | `ended`.

O transporte é imutável até `ended`. Evento desconhecido continua
seguro para clientes v2 antigos ignorarem.

### `presence`

```json
{
  "type": "presence",
  "payload": {
    "participantCount": 3
  }
}
```

Conta sessões de sala com WebSocket attached, como na feature anterior.
Mídia conectada não duplica presença.

### `media.state`

```json
{
  "type": "media.state",
  "payload": {
    "state": "connected",
    "role": "viewer",
    "mediaSessionId": "opaque-id",
    "transport": "websocket",
    "publicationId": "opaque-id"
  }
}
```

`state`: `connecting` | `connected` | `reconnecting` | `failed` | `closed`.

- Evento individual MUST ser enviado somente à sessão dona da mídia.
- Evento global do publisher MAY omitir `mediaSessionId` ao ser
  transmitido à sala.
- IDs são opacos; SDP, ICE candidate, endereço IP e token são proibidos.
- `transport` e `publicationId` são opcionais para compatibilidade com
  clientes v2 existentes.

## Ordering

1. Presenter recebe/produz `connecting` com transporte escolhido.
2. WebRTC `OnTrack(video)` ou WebSocket init + primeiro Cluster produz
   `publication.state=live` e `room.state=sharing`.
3. Viewer usa o transporte indicado: negocia subscriber WebRTC por HTTP
   ou solicita ticket e abre socket de mídia.
4. Viewer recebe `media.state=connected` e track/bootstrap.
5. Stop/queda produz `media.state=closed`,
   `publication.state=ended` e `room.state=waiting`.

Eventos podem ser repetidos; clientes devem tratá-los de modo idempotente.

## Isolation

- Eventos de um link MUST NOT ser enviados a sessão de outro link.
- Falha de subscriber não muda `room.state`.
- Falha/stop do publisher fecha todos os subscribers daquela sala.
