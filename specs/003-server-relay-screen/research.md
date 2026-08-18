# Research: 003-server-relay-screen

## 1. Topologia de mídia

**Decision**: dois transportes no processo Go, escolhidos por publicação.
WebRTC/UDP usa o SFU Pion existente; WebSocket usa WebM/VP8 binário
produzido por MediaRecorder e redistribuído sem transcodificação. Ambos
terminam no servidor e nunca conectam participantes.

**Rationale**: cumpre apresentador→servidor→viewer e a escolha explícita
entre UDP e porta HTTPS. O upload do presenter continua um fluxo e
nenhum modo adiciona serviço externo.

**Alternatives considered**:

- P2P/malha: proibido pelo requisito.
- Fallback automático: torna comportamento e diagnóstico imprevisíveis e
  contraria a escolha explícita.
- WebTransport/WebCodecs customizado: menor interoperabilidade e maior
  protocolo próprio que MediaRecorder/WebM.
- SFU externo (mediasoup/Janus/LiveKit): operacionalmente mais complexo e
  adiciona outro serviço/runtime.
- MCU/transcodificação: CPU alta e desnecessária para uma tela/codec.

## 2. Encaminhamento RTP

**Decision**: quando Pion recebe `TrackRemote` VP8 do presenter, criar uma
`TrackLocalStaticRTP` com a mesma capability e escrever nela cada pacote
RTP. Todos os subscribers vinculam essa track local. Ler RTCP de cada
sender e enviar PLI periódico ao publisher.

**Rationale**: é o padrão SFU oficial do ecossistema Pion; mantém payload
comprimido e isola a saída por PeerConnection. PLI recupera keyframe para
late joiners e após perda.

**Alternatives considered**:

- `TrackLocalStaticSample`: exige reconstruir samples/timestamps.
- Copiar bytes de UDP manualmente: perde DTLS/SRTP/ICE e interoperabilidade.
- Um encoder por viewer: viola simplicidade e escala inicial.

## 3. Codec

**Decision**: VP8 obrigatório nesta versão. O Angular aplica preferência
VP8 ao transceiver de vídeo; o MediaEngine Pion registra VP8 e feedback
RTCP necessário. Áudio não é registrado/adicionado.

**Rationale**: VP8 é suportado pelos navegadores desktop alvo e permite
forwarding homogêneo sem transcodificação. Restringir codec torna a
primeira implementação determinística.

**Alternatives considered**:

- Aceitar VP8/H264/VP9 dinamicamente: amplia matriz de teste e pode gerar
  incompatibilidade entre publisher e subscriber.
- H264 somente: perfis variam entre navegadores.
- Transcodificar: fora de escopo.

## 4. Negociação e versão do contrato

**Decision**: HTTP JSON `/api/v2`. Browser cria oferta, aguarda
`iceGatheringState=complete`, envia SDP ao endpoint publisher/subscriber e
recebe answer completa após o gathering do servidor. Sem trickle ICE no
WebSocket. Replicar os endpoints de link/sessão/start/stop em v2 e manter
v1 somente durante migração, mas v1 não deve continuar retransmitindo
`signal`.

**Rationale**: non-trickle elimina troca assíncrona de ICE e qualquer
roteamento participante→participante. Remover `signal/ready/to` é mudança
incompatível; MAJOR v2 atende à constituição.

**Alternatives considered**:

- Trickle ICE via WebSocket: menor setup em alguns cenários, mas mantém
  protocolo de sinalização mais complexo.
- Alterar v1 em lugar: viola versionamento obrigatório.
- SDP no WebSocket: funcionalidade de negociação deixa de ser contrato
  HTTP JSON.

## 5. Endpoints de mídia

**Decision**:

- `POST /api/v2/links/{id}/media/publisher`:
  `sessionId`, `presenterToken`, SDP offer → `mediaSessionId`, answer.
- `POST /api/v2/links/{id}/media/subscribers`:
  `sessionId`, SDP offer recvonly → `mediaSessionId`, answer.
- `DELETE /api/v2/links/{id}/media/subscribers/{mediaSessionId}`:
  teardown explícito; state callback também limpa queda abrupta.
- share stop existente fecha publisher e todos subscribers.

O estado `sharing` só é publicado após o servidor receber a track de
vídeo, não apenas após uma intenção HTTP.

**Rationale**: autorização reaproveita link/session/token; media session
IDs são opacos e isolam teardown. Viewer que entrou em waiting cria a
subscription ao receber `room.state=sharing`.

**Alternatives considered**:

- Publisher e subscriber no mesmo endpoint: papéis/autorização ambíguos.
- Marcar sharing antes de `OnTrack`: cria palco “transmitindo” sem mídia.
- Persistir media session: inútil após restart.

## 6. Rede e deploy

**Decision**: Pion `SettingEngine` em ICE Lite, UDP4 e
`SetICEUDPMux` sobre uma única `MEDIA_UDP_PORT` (default 5000).
`MEDIA_PUBLIC_IP` é opcional em localhost/LAN e obrigatório quando o
servidor está atrás de NAT; aplica candidate NAT 1:1. HTTP continua em
`PORT`. Produção usa HTTPS/WSS no proxy e libera UDP da mídia.

**Rationale**: uma porta UDP simplifica firewall/container. ICE Lite é
adequado ao endpoint servidor publicamente alcançável. DTLS-SRTP cifra
mídia mesmo com TLS terminado no proxy.

**Alternatives considered**:

- Faixa dinâmica UDP: firewall/deploy mais difícil.
- TURN como requisito: útil em redes restritivas, mas fora do escopo
  inicial; o servidor já é o endpoint público.
- WebRTC sobre TCP apenas: pior para mídia sob perda.

## 7. Capacidade e isolamento

**Decision**: `MEDIA_MAX_ROOMS` (default 20) e
`MEDIA_MAX_VIEWERS_PER_ROOM` (default 10, mínimo permitido 10). Manager
recusa antes de criar PeerConnection quando excedido. Cada subscriber
tem lifecycle próprio; write/RTCP/connection failure remove somente ele.

**Rationale**: servidor agora paga toda banda de saída. Limites explícitos
evitam exaustão silenciosa e atendem o caso mínimo de 10.

**Alternatives considered**:

- Sem limite: risco de memória/socket/banda.
- Fila de viewers: experiência imprevisível para transmissão ao vivo.
- Limite global fixo no código: difícil adequar ao host.

## 8. Eventos e recuperação

**Decision**: WebSocket v2 aceita nenhum frame de sinalização do cliente e
emite somente `room.state`, `presence` e `media.state`. O cliente usa
`RTCPeerConnection.connectionState` para `connecting/connected/failed`;
em `failed`, fecha a sessão e tenta renegociar com backoff limitado
enquanto a sala permanecer sharing.

**Rationale**: separa controle da mídia e torna reconexão observável.
Fechar publisher atualiza SQLite para waiting e encerra subscribers.

**Alternatives considered**:

- Manter `signal` ignorado: contrato ambíguo e superfície P2P residual.
- Reconexão infinita: loops e carga sem feedback.
- Congelar último frame: contradiz estado claro/sem histórico.

## 9. Privacidade e observabilidade

**Decision**: nenhum dump SDP/RTP, token ou endereço ICE remoto em logs.
Registrar somente link ID, media session ID, papel, transição de estado,
contadores e erro categorizado. Não gravar RTP em arquivo/SQLite.

**Rationale**: SDP pode revelar endereços; mídia e tokens são sensíveis.
IDs e estados bastam para diagnóstico operacional.

**Alternatives considered**:

- Logar SDP em debug: risco de exposição e não necessário.
- Métricas por conteúdo/frame: invasivo; contadores de pacotes/bytes são
  suficientes futuramente.

## 10. Testes

**Decision**: teste unitário do manager/capacidade; teste de integração
Pion com PeerConnections locais e RTP VP8 sintético; teste com 10
subscribers; integração WebM parser/socket; contratos Gin para
auth/SDP/ticket/erros; Vitest verifica os dois transportes. Matriz real
MediaRecorder→Go→MediaSource roda como gate pré-release em navegadores
suportados, mesmo que não rode em todo CI headless.

**Rationale**: prova a fronteira que falhou na implementação P2P e evita
depender de permissão real de captura nos testes automatizados.

**Alternatives considered**:

- Apenas mocks Pion: não prova passagem RTP.
- Depender somente de E2E com captura real em todo CI: permissões/hardware
  são frágeis; unitários/fixtures continuam necessários.

## 11. Seleção e compatibilidade de transporte

**Decision**: backend expõe ambos por padrão e valida configuração
`MEDIA_ALLOWED_TRANSPORTS`/`MEDIA_DEFAULT_TRANSPORT`. O Angular possui
lista/default de deploy e mostra ao presenter a interseção com
`GET /api/v2/media/config`. A publicação grava transporte imutável;
viewers o descobrem no snapshot/evento e usam o mesmo.

**Rationale**: evita UI anunciar opção indisponível e mantém escolha por
publicação sem persistir preferência no SQLite. Campos/eventos novos são
opcionais, portanto a extensão permanece aditiva em `/api/v2`.

**Alternatives considered**:

- Somente config Angular: pode divergir do backend.
- Escolha individual do viewer: gera mismatch e fluxo impossível.
- API v3: desnecessária enquanto endpoints WebRTC não mudarem.

## 12. Autorização do socket de mídia

**Decision**: endpoint HTTP JSON emite ticket opaco, curto (30s) e de uso
único após validar link, sessão, papel, token presenter, transporte e
capacidade. O browser usa apenas o ticket no upgrade WSS; o servidor
consome antes de aceitar frames.

**Rationale**: WebSocket do navegador não permite header Authorization
arbitrário. Ticket evita token presenter durável na URL e mantém a
decisão de autorização em contrato Gin testável.

**Alternatives considered**:

- Token presenter na query: vaza com facilidade em access logs.
- Token no primeiro frame: upgrade ocorre antes de autenticar e ocupa
  recurso não autorizado.
- Cookie: muda o modelo atual de token/sessionStorage.

## 13. WebM, framing e reprodução

**Decision**: `MediaRecorder` usa
`video/webm;codecs=vp8`, áudio desabilitado, bitrate limitado e timeslice
público default 250ms. Cada Blob não vazio vira uma mensagem binária.
Um parser WebM incremental no servidor separa init e Clusters completos,
timestamps e pontos de acesso/keyframes; viewers recebem init e uma
sequência contígua iniciada em random access para alimentar
`MediaSource`/`SourceBuffer`.

**Rationale**: base64/JSON aumenta bytes e CPU; Blob URLs sucessivas não
formam playback contínuo. Delimitar Clusters permite snapshot late-join
começar em fronteira válida mesmo que mensagens WebSocket não coincidam
com elementos WebM.

**Alternatives considered**:

- Tratar cada Blob como arquivo independente: chunks MediaRecorder não
  são garantidamente WebM completos.
- WebCodecs + protocolo de frames: complexo e fora da decisão do usuário.
- Parse/transmux no viewer: duplica lógica e não resolve buffer seguro no
  servidor.

**Risk**: MediaRecorder→MSE WebM precisa de aceite real em
Chrome/Edge/Firefox. Navegador sem MIME/APIs compatíveis falha claramente;
não troca de transporte.

## 14. Buffer e backpressure WebSocket

**Decision**: guardar um init segment por geração e ring de Clusters
limitado simultaneamente a 2000ms e 8MiB (configurável). Mensagem de
entrada tem máximo default 4MiB, pois `timeslice` não é limite rígido.
Cada viewer possui fila limitada; overflow fecha
somente o consumidor lento. Stop/falha/restart zeram bytes.

**Rationale**: late join em <5s exige bootstrap, mas limite duplo impede
memória ilimitada e não constitui gravação/replay. Filas isoladas impedem
um viewer de bloquear publisher ou outros viewers.

**Alternatives considered**:

- Histórico desde o início: viola privacidade/capacidade.
- Nenhum buffer: late join depende de futuro init/keyframe.
- Uma fila compartilhada: head-of-line blocking.

## 15. Canal, lifecycle e observabilidade

**Decision**: mídia WebSocket usa path próprio na mesma porta HTTP/HTTPS
e socket diferente de `room-events`. Estado `sharing` ocorre somente após
init + primeiro Cluster válidos. Reconnect cria nova geração no mesmo
transporte. Logs guardam apenas IDs opacos, transporte, estado e
contadores; query/ticket/token/WebM são proibidos.

**Rationale**: separar sockets isola backpressure. Mesma origem WSS reduz
firewall/TLS e atende o pedido sem porta TCP extra.

**Alternatives considered**:

- Misturar mídia no socket de eventos: presença pode ficar atrás de
  chunks e buffers.
- Porta TCP dedicada: configuração/proxy desnecessários.
- Fallback para WebRTC: contraria seleção e mascara falhas.

## 16. Matriz de navegadores e keyframes

**Decision**: WebRTC permanece suportado em Chrome, Edge e Firefox.
WebSocket MediaRecorder/MSE entra inicialmente somente em versões
Chrome/Edge aprovadas por teste real. O Angular exige suporte de MIME/APIs
e uma matriz de versão validada antes de mostrar `websocket`. Firefox não
recebe essa opção enquanto não produzir keyframe VP8 dentro da janela de
2 segundos. Não há seleção/fallback silencioso.

**Rationale**: blobs/timeslice não garantem fronteira reproduzível e o
controle `videoKeyFrameIntervalDuration` é apenas recomendação da
especificação. Firefox pode espaçar keyframes por 6–10s; sem keyframe no
ring de 2s, é impossível garantir late join <5s sem aumentar histórico,
transcodificar ou mudar para WebCodecs, todos proibidos.

**Alternatives considered**:

- Aumentar buffer até o keyframe: contradiz FR-021.
- Esperar próximo keyframe: contradiz SC-002/SC-010.
- Transcodificar ou WebCodecs: fora de escopo.
- Expor WebSocket e falhar depois: experiência insegura e não testável.
