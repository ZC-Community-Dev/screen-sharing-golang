# Room Events (WebSocket)

**Path**: `GET /api/v1/links/{id}/events`

Upgrade WebSocket após a sessão HTTP existir. Query obrigatória:
`sessionId` retornado por `presenter-sessions` ou `viewer-sessions`.

O servidor MUST recusar upgrade se `id` for inválido/desconhecido ou se
`sessionId` não pertencer a esse link. Frames MUST ser JSON texto.
Frames MUST NOT conter `presenterToken`, hash de token ou `LINK_ID_SALT`.

## Client → Server

### `signal`

Encaminha SDP ou ICE para outro peer da mesma sala.

```json
{
  "type": "signal",
  "to": "<sessionId destino>",
  "payload": {
    "kind": "offer",
    "sdp": "<sdp>"
  }
}
```

`kind`: `offer` | `answer` | `ice` | `ready`.

Para `ice`, `payload.candidate` substitui `sdp`.

`to` aceita um `sessionId` da sala **ou** o alias `"presenter"`.
O servidor MUST reescrever `"presenter"` para o `sessionId` do
apresentador ativo. Se não houver apresentador, MUST descartar o frame.

O servidor MUST entregar só se `to` estiver na mesma sala. MUST NOT
retransmitir a clientes de outro `id`.

### Handshake de late-join (obrigatório)

1. Espectador recebe `room.state` com `state=sharing` (no connect ou
   depois do start) e envia:

```json
{
  "type": "signal",
  "to": "presenter",
  "payload": { "kind": "ready" }
}
```

2. O apresentador MUST responder com `offer` para o `from` desse
   `ready` (um `RTCPeerConnection` por espectador).
3. Espectador responde `answer`; ambos trocam `ice`.
4. Espectador MUST NOT enviar `offer` de captura.
5. O cliente MUST enfileirar `signal` até o WebSocket estar `OPEN`.

## Server → Client

### `room.state`

```json
{
  "type": "room.state",
  "payload": {
    "state": "waiting"
  }
}
```

`state`: `waiting` | `sharing`. Enviado no connect e a cada start/stop
ou queda do apresentador.

### `presence`

```json
{
  "type": "presence",
  "payload": {
    "participantCount": 3
  }
}
```

Enviado no connect e quando uma sessão entra ou sai.

### `signal`

Mesmo formato do cliente, com `from` em vez de `to`:

```json
{
  "type": "signal",
  "from": "<sessionId origem>",
  "payload": {
    "kind": "answer",
    "sdp": "<sdp>"
  }
}
```

## Regras de mídia (fora do JSON)

- O apresentador cria um `RTCPeerConnection` por espectador (malha).
- `getDisplayMedia({ video: true, audio: false })`.
- Nenhuma trilha de áudio MUST ser adicionada.
- Espectador só recebe; MUST NOT enviar `offer` de captura de tela.
