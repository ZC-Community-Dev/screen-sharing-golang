# Screen Share

Compartilhamento de tela por link, com Angular no navegador e API
Go/Gin. Toda mídia passa pelo servidor, sem conexão P2P. O apresentador
pode usar WebRTC/UDP via SFU Pion ou WebSocket/WebM na mesma origem
HTTPS; cada publicação mantém um único transporte.

## Desenvolvimento

Use dois terminais para manter hot reload:

```powershell
# terminal 1
cd api
if (-not (Test-Path .env)) { Copy-Item .env.example .env }
go run ./cmd/server

# terminal 2
cd app
npm start
```

Abra `http://127.0.0.1:4200`. O Angular encaminha `/api` e WebSocket
para a API em `http://127.0.0.1:8080`.

## Build de produção (um processo)

```powershell
cd app
npm test
npm run test:build-script
npm run build

cd ../api
go test ./...
go build -o screen-share.exe ./cmd/server
./screen-share.exe
```

Abra `http://127.0.0.1:8080`. O `npm run build`:

1. compila o Angular em `app/dist/app/browser/`;
2. limpa e copia o bundle para `api/internal/web/dist/browser/`;
3. permite que `go:embed` incorpore a interface no executável.

O executável não precisa dos arquivos Angular ao lado em runtime. Ele
ainda precisa das configurações da API (por exemplo `api/.env`) e cria
ou abre o SQLite conforme `LINKS_DB_PATH`.

## Configuração e segredos

- Configuração pública do frontend fica em `app/src/environments/`.
- `LINK_ID_SALT`, banco e tokens permanecem somente na API.
- `CORS_ORIGINS` lista origens extras (dev Angular e o HTTPS público
  atrás do Cloudflare). Same-origin (`https://example.local`) é
  aceito automaticamente quando o `Host` público chega ao processo Go.
- O pipeline rejeita `.env` e bancos dentro do bundle do frontend.
- `MEDIA_UDP_PORT` (padrão `5000`) deve aceitar UDP no firewall.
- `MEDIA_UDP_MTU` (padrão `1200`) é o tamanho máximo do datagrama UDP de
  mídia. Valores maiores fragmentam atrás de internet/VPN e deixam a
  tela cinza. O frontend usa o mesmo teto em `mediaUdpMtu`.
- `MEDIA_PUBLIC_IP` fica vazio em localhost/LAN; atrás de NAT, configure o
  IP público cujo UDP é encaminhado para `MEDIA_UDP_PORT`.
- No frontend, `mediaUdpHost`, `mediaUdpPort` e `mediaUdpMtu` em
  `app/src/environments/` apontam o navegador para o IPv4/porta UDP e
  limitam cada pacote a 1200 bytes. `mediaUdpHost` vazio preserva o SDP.
- `MEDIA_MAX_ROOMS` e `MEDIA_MAX_VIEWERS_PER_ROOM` limitam capacidade; o
  limite por sala deve ser no mínimo 10.
- `MEDIA_ALLOWED_TRANSPORTS=webrtc,websocket` e
  `MEDIA_DEFAULT_TRANSPORT=webrtc` controlam o backend. O frontend define
  sua lista/default em `app/src/environments/` e mostra somente opções
  aceitas pelos dois lados.
- `MEDIA_WS_MAX_CHUNK_BYTES` e `MEDIA_WS_MAX_BUFFER_BYTES` limitam
  mensagens e o buffer WebSocket, que nunca excede 2 segundos e é
  apagado ao parar/falhar.
- Em produção, termine HTTPS/WSS em proxy reverso. SDP, tokens,
  tickets/query, endereços ICE e conteúdo RTP/WebM não devem ser
  registrados.

Veja os cenários completos em
`specs/003-server-relay-screen/quickstart.md`.

## Build Linux

Compile o frontend antes para incorporá-lo e exponha tanto a porta HTTP
TCP quanto a porta de mídia UDP:

```powershell
cd app
npm ci
npm run build

cd ../api
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o screen-share-linux-amd64 ./cmd/server
```

O binário continua usando `api/.env` (ou variáveis do ambiente) em
runtime. Não exponha o SQLite, `LINK_ID_SALT` ou tokens no frontend.

## Rede e navegadores

- Encaminhe HTTP e upgrades WSS de eventos/mídia para a mesma porta do
  processo Go. No Cloudflare, ative WebSockets (Network) e use SSL Full
  ou Full (strict). O handshake WSS aceita same-origin; se o `Host`
  interno for o IP de origem, defina `CORS_ORIGINS` com a origem HTTPS
  pública (`https://example.local`). O proxy da Cloudflare **não** encaminha
  UDP: em domínio público o default de mídia deve ser `websocket`
  (`defaultMediaTransport` no frontend e, se quiser, `MEDIA_DEFAULT_TRANSPORT`).
  WebRTC só funciona se o navegador alcançar `mediaUdpHost:MEDIA_UDP_PORT`
  direto (LAN ou UDP aberto no IP público), sem passar pela Cloudflare.
- Libere `MEDIA_UDP_PORT` somente quando `webrtc` estiver habilitado.
- Chrome/Edge validados oferecem WebRTC e WebSocket.
- Firefox permanece em WebRTC nesta versão; a opção WebSocket fica
  oculta até garantir keyframes VP8 dentro da janela de 2 segundos.
- Não há fallback automático: falhas são exibidas e a publicação mantém
  o transporte escolhido.
