# Media WebSocket v2

Canal binário servidor-mediado para publicações configuradas com
`transport=websocket`. Ele é separado de `room-events-v2.md`; mídia lenta
ou volumosa nunca pode bloquear presença e estado.

## Upgrade e autenticação

1. O cliente solicita um ticket em
   `POST /api/v2/links/{id}/media/websocket-tickets`.
2. Em até 30 segundos, abre
   `GET /api/v2/links/{id}/media/websocket?ticket={opaque}` com o
   subprotocolo `screen-share-media-v1`.
3. O servidor valida origem, link, papel, sessão, capacidade e consome o
   ticket antes de aceitar mídia. Ticket expirado/reutilizado é recusado.

O valor bruto do ticket, query string, token, IP remoto e payload nunca
podem aparecer em logs. A conexão usa `ws://` em desenvolvimento e
`wss://` na mesma origem HTTPS em produção.

## Publisher

O ticket contém papel `publisher` e já foi emitido somente após validar o
token de apresentador. Após o upgrade, o cliente envia primeiro:

```json
{
  "type": "publisher.open",
  "protocolVersion": 1,
  "mimeType": "video/webm;codecs=vp8",
  "timesliceMs": 250
}
```

O servidor responde:

```json
{
  "type": "media.opened",
  "publicationId": "opaque-id",
  "mediaSessionId": "opaque-id",
  "transport": "websocket",
  "startupBufferMs": 2000,
  "maxChunkBytes": 4194304
}
```

Cada mensagem binária seguinte contém exatamente um `Blob` produzido
pelo `MediaRecorder`. O servidor trata as mensagens como um stream WebM
ordenado; fronteiras de mensagem não são segmentos. Um parser incremental
extrai init, Clusters completos, timestamps e random-access/keyframes. A
sala só entra em `sharing` após init válido e primeiro Cluster.

Texto adicional aceito do publisher:

- `media.reset`: inicia nova geração WebM; esvazia init/ring anterior.
- `media.end`: encerra normalmente a publicação.

Texto desconhecido, áudio, MIME diferente, sequência WebM inválida ou
frame acima do limite encerra somente a publicação com erro seguro.

## Viewer

O ticket contém papel `viewer`, sessão da mesma sala e publication ID
ativo. Após o upgrade, o cliente envia:

```json
{
  "type": "subscriber.open",
  "protocolVersion": 1
}
```

O servidor responde `media.opened` e envia:

1. `media.bootstrap` texto com `generation` e quantidade de Clusters;
2. uma mensagem binária com init segment;
3. sequência contígua iniciada no random-access Cluster mais recente,
   cobrindo no máximo 2 segundos;
4. `media.live` texto;
5. novos Clusters binários ao vivo.

O viewer anexa init/Clusters em ordem a um `SourceBuffer` com MIME
`video/webm;codecs=vp8`. Em `media.reset`, limpa o `MediaSource` antigo e
aguarda novo bootstrap.

## Buffer e backpressure

- O ring é limitado simultaneamente a 2000 ms e ao limite configurado em
  bytes; o menor limite prevalece.
- Snapshot late-join é atômico e não duplica Cluster entre bootstrap e
  fluxo ao vivo.
- Stop, falha terminal, perda do presenter e restart apagam todos os
  bytes imediatamente.
- Cada viewer possui fila limitada própria. Overflow fecha somente esse
  viewer com `media_slow_consumer`; publisher e demais viewers continuam.
- Não existe endpoint de download, replay ou consulta ao buffer.

## Controle, keepalive e fechamento

Mensagens JSON servidor→cliente:

- `media.opened`
- `media.bootstrap`
- `media.live`
- `media.reset`
- `media.end`
- `media.error` com código seguro

Ping/pong do protocolo mantém a conexão; não usar frames JSON como
heartbeat. Após timeout ou ausência de chunks, o servidor encerra a
sessão e publica a transição correspondente.

Close codes de aplicação:

- `4400`: `media_protocol_error`
- `4401`: `media_unauthorized`
- `4404`: `media_session_not_found`
- `4409`: `publication_transport_mismatch` ou conflito
- `4429`: `media_slow_consumer`

Frame grande usa o código padrão `1009`. Conteúdo de mídia, ticket e
token nunca entram na reason de fechamento.

## Compatibilidade

Endpoints WebRTC v2 permanecem inalterados. Uma publicação escolhe
exatamente um transporte e não troca automaticamente. Cliente v2 antigo
ignora novos eventos, mas não consegue assistir publicação WebSocket;
frontend e backend devem ser implantados juntos antes de habilitá-la.

O modo WebSocket requer matriz de navegador aprovada. Nesta versão,
Chrome/Edge validados podem anunciá-lo; Firefox usa WebRTC quando
disponível até garantir keyframe VP8 dentro da janela de 2 segundos.
