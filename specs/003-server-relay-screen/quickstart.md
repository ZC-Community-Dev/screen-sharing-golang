# Quickstart: 003-server-relay-screen

Validação ponta a ponta do relay de tela pelo servidor.

- HTTP: [contracts/http-api-v2.yaml](./contracts/http-api-v2.yaml)
- Config frontend: [contracts/frontend-media-config.md](./contracts/frontend-media-config.md)
- Eventos: [contracts/room-events-v2.md](./contracts/room-events-v2.md)
- Mídia WebSocket: [contracts/media-websocket-v2.md](./contracts/media-websocket-v2.md)
- Migração: [contracts/migration-v1-to-v2.md](./contracts/migration-v1-to-v2.md)
- Modelo: [data-model.md](./data-model.md)

## Prerequisites

- Go 1.25+
- Node.js/npm compatível com Angular 21
- Chrome/Edge desktop para os dois transportes
- Firefox desktop para WebRTC; WebSocket fica oculto até aprovação da
  matriz real de keyframes/MediaSource
- Porta HTTP TCP 8080
- Porta de mídia UDP 5000 liberada para testar `webrtc`
- Proxy/servidor aceitando upgrade WebSocket na porta HTTP para testar
  `websocket`
- Dois ou mais perfis de navegador

## Configuração local

`api/.env`:

```dotenv
LINK_ID_SALT=replace-with-a-long-random-secret
LINKS_DB_PATH=data/links.db
PORT=8080
MEDIA_UDP_PORT=5000
MEDIA_PUBLIC_IP=
MEDIA_MAX_ROOMS=20
MEDIA_MAX_VIEWERS_PER_ROOM=10
MEDIA_ALLOWED_TRANSPORTS=webrtc,websocket
MEDIA_DEFAULT_TRANSPORT=webrtc
MEDIA_WS_MAX_CHUNK_BYTES=4194304
MEDIA_WS_MAX_BUFFER_BYTES=8388608
```

Em localhost/LAN, `MEDIA_PUBLIC_IP` pode ficar vazio. Em servidor atrás
de NAT, usar o IP público encaminhado para `MEDIA_UDP_PORT`.

No Angular, `environment.ts` define a lista/default exibidos:

```typescript
allowedMediaTransports: ['webrtc', 'websocket'],
defaultMediaTransport: 'webrtc',
```

O backend continua autoritativo; o frontend mostra apenas a interseção
entre sua configuração de deploy, `GET /api/v2/media/config` e a matriz
de navegador validada. Ausência da opção não provoca fallback.

## Testes

```powershell
cd api
go test ./...

cd ../app
npm test
npm run build
```

Esperado:

- unitários de capacidade/lifecycle passam;
- integração Pion prova RTP publisher→server→subscriber;
- integração WebSocket prova WebM publisher→buffer→10 viewers;
- framing inválido, chunk grande e viewer lento são isolados;
- late join recebe init + no máximo 2 segundos e continua ao vivo;
- teste com 10 subscribers passa;
- contratos v2 e testes Angular passam;
- não há teste/código que roteie `signal`, `ready`, `to` ou `from`.

## Executar

```powershell
cd api
go run ./cmd/server
```

Abrir `http://127.0.0.1:8080`. O frontend de produção já está embutido
quando `npm run build` foi executado antes do build Go.

## Cenário 1: publisher WebRTC/UDP

1. Gerar um link e entrar como presenter.
2. Iniciar compartilhamento de tela.
3. Confirmar estados `connecting` → `sharing`.
4. Abrir 3 viewers.
5. Inspecionar `chrome://webrtc-internals` (ou equivalente).

Esperado:

- presenter possui uma PeerConnection de mídia, com remote endpoint do
  servidor;
- novos viewers não criam peers/tracks adicionais no presenter;
- nenhum SDP/candidate contém session ID de viewer.

## Cenário 2: viewer recebe do servidor

1. Com sharing ativo, abrir convite noutro perfil.
2. Confirmar tela em <5s.
3. Inspecionar conexão do viewer.

Esperado:

- viewer possui uma PeerConnection recvonly com o servidor;
- não há endereço/oferta do presenter;
- fechar viewer reduz presença e não afeta outro viewer.

## Cenário 3: publisher e viewers WebSocket

1. Habilitar ambos os transportes e selecionar `WebSocket` na tela.
2. Iniciar a captura e confirmar `connecting` → `sharing`.
3. Abrir viewer e inspecionar DevTools/Network.
4. Confirmar socket de mídia binário separado do socket de eventos.

Esperado:

- presenter usa `MediaRecorder` WebM/VP8 e nenhuma PeerConnection;
- viewer usa `MediaSource` e nenhuma PeerConnection;
- ambos conectam somente à mesma origem do backend;
- query/token/chunks não aparecem em logs;
- UDP pode estar bloqueado sem afetar este cenário.
- repetir late join em posições aleatórias e confirmar início por
  random-access Cluster dentro da janela de 2 segundos.

## Cenário 4: 10 viewers nos dois transportes

1. Executar uma vez com `webrtc` e outra com `websocket`.
2. Abrir 10 sessões viewer no mesmo link.
3. Confirmar que todas recebem a mídia.
4. Fechar uma sessão e continuar medindo as demais.

Esperado:

- publisher mantém uma entrada;
- servidor mantém 10 subscribers isolados;
- saída de um não interrompe os outros por >1s;
- tentativa do 11º viewer respeita o limite configurado com erro claro,
  se o limite for 10.

## Cenário 5: late join e buffer WebSocket

1. Manter publicação WebSocket ativa por mais de 10 segundos.
2. Abrir um novo viewer.
3. Medir bootstrap e encerrar a publicação.

Esperado:

- tela aparece em <5s;
- bootstrap contém init e no máximo 2 segundos recentes;
- viewer passa para dados ao vivo sem duplicação;
- após stop/restart nenhum byte fica disponível.

## Cenário 6: autorização e isolamento

1. Tentar endpoint publisher com sessão viewer.
2. Tentar subscriber com session ID de outro link.
3. Enviar frame `signal` ao WebSocket v2.
4. Reusar ticket de mídia ou abrir viewer com publicação de outro link.

Esperado:

- publicação é 401/403;
- cross-room é recusado;
- frame P2P não é retransmitido;
- ticket reutilizado/cross-room é recusado;
- transmissão válida continua.

## Cenário 7: stop, queda e restart

1. Parar o share: todos voltam a waiting em <5s.
2. Iniciar novamente com o mesmo link/token.
3. Derrubar rede/aba do presenter.
4. Reiniciar API durante sharing.

Esperado:

- subscribers fecham e não exibem frame congelado indefinidamente;
- ninguém vira presenter;
- nenhuma reprodução antiga existe;
- após restart o link abre em waiting e pode publicar novamente.

## Produção

- Terminar HTTPS/WSS em proxy reverso.
- Encaminhar upgrade do socket de eventos e do socket binário de mídia.
- Encaminhar UDP `MEDIA_UDP_PORT` quando `webrtc` estiver habilitado.
- Configurar `MEDIA_PUBLIC_IP` quando houver NAT.
- Não registrar SDP, ICE remoto, token ou conteúdo.
- Monitorar rooms/subscribers ativos, bytes/pacotes e transições de erro.
