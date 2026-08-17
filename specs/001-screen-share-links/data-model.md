# Data Model: 001-screen-share-links

## Link

Convite público persistido. Fonte de verdade: SQLite.

| Field | Type | Rules |
|---|---|---|
| `id` | string | PK. Base62 (`0-9A-Za-z`), comprimento ≥ 8 (gerado com 10). Único. Imprevisível (HMAC + sal). |
| `presenter_token_hash` | string | SHA-256 hex do token em claro. Nunca retornado em API. |
| `created_at` | datetime (UTC ISO-8601) | Preenchido na criação. Imutável. |
| `state` | enum | `waiting` \| `sharing`. Default `waiting`. Ao arrancar o processo, qualquer `sharing` volta a `waiting`. |

**Validation**:

- `id` MUST casar `^[0-9A-Za-z]{8,}$`. Fora disso → link inválido, sem INSERT.
- Colisão de `id` → gerar outro, não reutilizar.
- `presenter_token_hash` MUST NOT ser nulo.

**State transitions**:

```text
(create) --> waiting
waiting  -- presenter start share (token ok, sem outro apresentador ativo) --> sharing
sharing  -- presenter stop share OR presenter disconnect OR process restart --> waiting
```

Não há estado `invalid` persistido: id desconhecido é 404, não uma linha.

## PresenterToken

Credencial de apresentação. Não é tabela própria.

| Field | Type | Rules |
|---|---|---|
| `value` | string | Base62, 32 bytes aleatórios encodados. Emitido uma vez em `POST /links`. |
| `hash` | string | `SHA-256(value)` persistido em `Link.presenter_token_hash`. |

O cliente apresentador guarda `value` em `sessionStorage` (`presenterToken:<id>`).
Comparação no servidor é sempre hash. Token de outro link MUST falhar o claim.

## Session (em memória)

Presença temporária numa sala. Morre com o processo ou com o WebSocket.

| Field | Type | Rules |
|---|---|---|
| `session_id` | string | Opaco, único no processo. |
| `link_id` | string | FK lógica para `Link.id`. |
| `role` | enum | `presenter` \| `viewer`. |
| `connected_at` | datetime | UTC. |

**Rules**:

- No máximo uma sessão `presenter` ativa por `link_id`.
- Várias sessões `viewer`.
- Claim de apresentador com token válido e apresentador já ativo → 409.
- Queda da sessão apresentador → `Link.state = waiting` e broadcast aos viewers.
- `participantCount` = sessões **com WebSocket attached** (`send != nil`),
  não o join HTTP sozinho.
- Join HTTP sem attach em 30s MUST ser descartado (não conta, não bloqueia).

## RoomEvent (mensagem de tempo real)

Não persistido. Contrato em `contracts/room-events.md`.

| Field | Type | Rules |
|---|---|---|
| `type` | enum | `presence` \| `room.state` \| `signal` |
| `link_id` | string | Sala alvo. |
| `payload` | object | Conforme o tipo. MUST NOT conter token nem sal. |

## ScreenTransmission

Fluxo WebRTC. Não persistido.

| Field | Type | Rules |
|---|---|---|
| `link_id` | string | Um fluxo ativo por link. |
| `from_session` | string | MUST ser o apresentador. |
| `to_session` | string | Um peer por espectador (malha). |
| `media` | constraint | Somente vídeo. Áudio MUST NOT ser adicionado à connection. |

## Relationships

```text
Link 1 -- * Session (memória)
Link 1 -- 0..1 ScreenTransmission (memória, só se state=sharing)
Link 1 -- 1 PresenterToken (hash persistido, valor só no cliente criador)
```

## Persistence notes

- Arquivo SQLite: `api/data/links.db`.
- Índices: PK em `id` basta.
- Sem tabela de usuários, chat ou gravação.
- Segredos (`LINK_ID_SALT`, token em claro) MUST NOT ir para log nem para
  respostas de espectador.
