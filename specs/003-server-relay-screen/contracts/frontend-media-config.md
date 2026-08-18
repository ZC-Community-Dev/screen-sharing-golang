# Frontend Media Configuration

Configuração pública de deploy em
`app/src/environments/environment*.ts`. Não contém token, salt nem
limites operacionais privados do processo Go.

```typescript
type MediaTransport = 'webrtc' | 'websocket';

interface PublicMediaEnvironment {
  allowedMediaTransports: readonly MediaTransport[];
  defaultMediaTransport: MediaTransport;
  mediaUdpHost: string;
  mediaUdpPort: number;
  mediaUdpMtu: number;
}
```

Exemplo:

```typescript
allowedMediaTransports: ['webrtc', 'websocket'],
defaultMediaTransport: 'webrtc',
mediaUdpHost: '192.168.10.108',
mediaUdpPort: 5000,
mediaUdpMtu: 1200,
```

`mediaUdpHost` vazio preserva o SDP do servidor. Quando preenchido com um
IPv4, o cliente reescreve `c=` e candidates `typ host` da resposta ICE
Lite para esse endereço e `mediaUdpPort` (que deve coincidir com
`MEDIA_UDP_PORT`). `mediaUdpMtu` (padrão 1200) é o teto do datagrama UDP;
o cliente limita bitrate/packetização para não fragmentar. Isso não cria
STUN/TURN nem conexão P2P.

## Validação

- A lista deve ser não vazia, única e conter apenas valores conhecidos.
- O default deve pertencer à lista.
- O cliente busca `GET /api/v2/media/config` e calcula a interseção.
- `websocket` também exige
  `MediaRecorder.isTypeSupported('video/webm;codecs=vp8')`,
  `MediaSource.isTypeSupported('video/webm;codecs="vp8"')` e browser
  presente na matriz Chrome/Edge aprovada.
- Opção indisponível fica oculta/desabilitada com explicação; nunca
  inicia fallback automático.

## Seleção na sala

- Com uma opção disponível, ela é selecionada sem diálogo adicional.
- Com duas, o presenter escolhe antes de `getDisplayMedia`.
- A escolha vale somente para a publicação atual e não é salva no
  SQLite/localStorage.
- Viewers não escolhem: usam `publication.transport` anunciado pelo
  backend ou exibem erro incompatível.
- Stop permite ao presenter iniciar nova publicação e escolher novamente.
