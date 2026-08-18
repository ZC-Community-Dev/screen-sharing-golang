# Frontend Media Configuration

Configuração pública de deploy em
`app/src/environments/environment*.ts`. Não contém token, IP público,
porta UDP nem limite operacional privado.

```typescript
type MediaTransport = 'webrtc' | 'websocket';

interface PublicMediaEnvironment {
  allowedMediaTransports: readonly MediaTransport[];
  defaultMediaTransport: MediaTransport;
}
```

Exemplo:

```typescript
allowedMediaTransports: ['webrtc', 'websocket'],
defaultMediaTransport: 'webrtc',
```

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
