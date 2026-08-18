# Data Model: 003-server-relay-screen

Links e tokens persistidos continuam conforme
`../001-screen-share-links/data-model.md`. As entidades abaixo existem
somente em memória no processo e nunca entram no SQLite.

## MediaRoom

Estado de mídia servidor-mediada de um link.

| Field | Type | Rules |
|---|---|---|
| `link_id` | string | Chave; link Base62 persistido deve existir |
| `publisher` | PublisherSession? | No máximo um |
| `transport` | enum? | `webrtc` ou `websocket`; imutável durante a publicação |
| `relay_track` | RTP track? | Criada quando o servidor recebe vídeo |
| `websocket_buffer` | WebSocketBuffer? | Somente no transporte WebSocket |
| `subscribers` | map&lt;id, SubscriberSession&gt; | Até o limite configurado |
| `state` | enum | `waiting`, `connecting`, `sharing`, `reconnecting`, `failed` |
| `created_at` | datetime | Para observabilidade; não persistido |

**Transitions**:

```text
waiting -- publisher WebRTC offer or WS upgrade --> connecting
connecting -- video OnTrack or WebM init+Cluster --> sharing
connecting -- timeout/failure --> failed --> waiting
sharing -- publisher disconnected --> reconnecting
reconnecting -- publisher restored --> sharing
reconnecting -- timeout/stop --> waiting
sharing -- explicit stop --> waiting
```

Entrar/sair subscriber não altera `sharing`.

## PublisherSession

Conexão navegador apresentador → servidor.

| Field | Type | Rules |
|---|---|---|
| `id` | string | Opaco e único no processo |
| `link_id` | string | Mesma sala autorizada |
| `room_session_id` | string | MUST ser presenter ativo do link |
| `transport` | enum | `webrtc` ou `websocket`; escolhido antes de publicar |
| `peer_connection` | server WebRTC peer? | Somente no modo WebRTC |
| `media_socket` | WebSocket? | Somente no modo WebSocket |
| `remote_track` | video track? | Somente WebRTC/VP8 |
| `mime_type` | string? | WebSocket exige `video/webm;codecs=vp8` |
| `state` | enum | `new`, `connecting`, `connected`, `failed`, `closed` |
| `created_at` | datetime | Efêmero |

**Rules**:

- Exige session ID presenter e token válido.
- Um publisher por `link_id`.
- Nunca contém subscriber IDs no SDP/controle.
- O transporte não muda durante a publicação.
- Fechar publisher fecha relay/buffer e todos subscribers da sala.

## RelayTrack

Trilha RTP local do SFU alimentada pelo publisher WebRTC. Não existe no
modo WebSocket.

| Field | Type | Rules |
|---|---|---|
| `codec` | enum | VP8 nesta versão |
| `track_id` | string | ID estável por publicação |
| `stream_id` | string | Derivado de media session, não de segredo |
| `last_packet_at` | datetime | Detectar fluxo parado |

Pacotes são encaminhados em memória. Não há histórico, arquivo nem
buffer reproduzível após fechamento.

## SubscriberSession

Conexão servidor → navegador viewer.

| Field | Type | Rules |
|---|---|---|
| `id` | string | `mediaSessionId` opaco |
| `link_id` | string | Sala solicitada |
| `room_session_id` | string | MUST ser sessão viewer conectada |
| `transport` | enum | MUST ser igual ao publisher da sala |
| `peer_connection` | server WebRTC peer? | Somente no modo WebRTC |
| `media_socket` | WebSocket? | Somente no modo WebSocket |
| `sender` | RTP sender? | Somente WebRTC; vinculado ao RelayTrack |
| `send_queue` | bounded queue? | Somente WebSocket; overflow encerra apenas o viewer |
| `state` | enum | `new`, `connecting`, `connected`, `failed`, `closed` |
| `created_at` | datetime | Efêmero |

**Rules**:

- Não aceita track de entrada.
- Não pode apontar para RelayTrack de outro link.
- Não pode usar transporte diferente da publicação.
- Falha/saída remove apenas esta sessão.
- RTCP deve ser consumido para manter o sender saudável.

## WebSocketBuffer

Bootstrap efêmero de uma publicação WebSocket.

| Field | Type | Rules |
|---|---|---|
| `generation` | integer | Incrementa quando o stream WebM reinicia |
| `init_segment` | bytes | Metadados WebM necessários ao MediaSource |
| `records` | ring of binary records | Ordem monotônica; somente dados recentes |
| `duration_ms` | integer | MUST ser ≤2000 |
| `size_bytes` | integer | MUST respeitar limite configurado |
| `last_record_at` | datetime | Detecta publicação parada |

O snapshot de late join é atômico: init da geração atual, registros
recentes reproduzíveis e depois fluxo ao vivo sem duplicação. Stop,
falha terminal e restart removem todos os bytes imediatamente.

## WebSocketTicket

Credencial curta, opaca e de uso único emitida por HTTP antes do upgrade.

| Field | Type | Rules |
|---|---|---|
| `id_hash` | bytes | Somente hash em memória; valor bruto nunca logado |
| `link_id` | string | Sala autorizada |
| `room_session_id` | string | Sessão presenter ou viewer |
| `role` | enum | `publisher` ou `viewer` |
| `transport` | enum | Sempre `websocket` |
| `expires_at` | datetime | TTL curto, recomendado 30 segundos |
| `consumed_at` | datetime? | Primeiro upgrade consome; replay é recusado |

## MediaTransportConfig

Configuração pública/operacional sem segredos.

| Field | Type | Rules |
|---|---|---|
| `allowed_transports` | set | Subconjunto não vazio de `webrtc`, `websocket` |
| `default_transport` | enum | MUST pertencer a `allowed_transports` |
| `websocket_timeslice_ms` | integer | Default 250; limite validado |
| `websocket_buffer_ms` | integer | Fixo/máximo 2000 |
| `websocket_max_chunk_bytes` | integer | Default 4 MiB; timeslice não limita Blob |
| `websocket_max_buffer_bytes` | integer | Default 8 MiB |

## MediaCapacity

Configuração e contadores do processo.

| Field | Type | Rules |
|---|---|---|
| `max_rooms` | integer | Default 20; >0 |
| `max_viewers_per_room` | integer | Default 10; MUST ser ≥10 |
| `active_rooms` | integer | Salas com publisher |
| `active_subscribers` | integer | Soma das distribuições |

Checagem de capacidade ocorre antes de alocar PeerConnection. Excesso
retorna erro categorizado e não remove sessão ativa.

## SDPExchange

Payload efêmero de negociação HTTP.

| Field | Type | Rules |
|---|---|---|
| `type` | string | Entrada `offer`; saída `answer` |
| `sdp` | string | Obrigatório, limite de tamanho, nunca logado |
| `session_id` | string | Sessão da sala |
| `presenter_token` | string? | Somente publisher; nunca persistido/logado |

## Relationships

```text
Link 1 ------ 0..1 MediaRoom
MediaRoom 1 - 0..1 PublisherSession
PublisherSession 1 -- 0..1 RelayTrack
PublisherSession 1 -- 0..1 WebSocketBuffer
MediaRoom 1 - 0..N SubscriberSession
RelayTrack 1 -- 0..N SubscriberSession
WebSocketBuffer 1 -- 0..N SubscriberSession
RoomSession 1 -- 0..N WebSocketTicket
```

## Restart behavior

No restart:

- todas as entidades desta feature desaparecem;
- links continuam no SQLite;
- qualquer link persistido como `sharing` volta a `waiting`;
- apresentador deve capturar/publicar novamente.
