# Screen Share

Compartilhamento de tela por link, com Angular no navegador e API
Go/Gin. A mídia é WebRTC P2P; o servidor controla links, presença e
sinalização.

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
- O pipeline rejeita `.env` e bancos dentro do bundle do frontend.

Veja os cenários completos em
`specs/002-embed-frontend/quickstart.md`.
