# Quickstart: 002-embed-frontend

Guia de validação do frontend Angular incorporado no serviço Go.
Contrato: [contracts/static-web.md](./contracts/static-web.md).
Modelo: [data-model.md](./data-model.md).

## Prerequisites

- Go 1.25+
- Node.js + npm compatíveis com Angular 21
- `api/.env` com `LINK_ID_SALT` não vazio
- Portas 8080 e, somente em desenvolvimento, 4200 livres

## 1. Testes antes do build

```powershell
cd app
npm test

cd ../api
go test ./...
```

Esperado: testes Angular e Go passam. Os testes Go do pacote web usam
um filesystem de fixture e não exigem build real do Angular.

## 2. Produzir o bundle incorporável

```powershell
cd app
npm run build
```

Esperado:

- Angular produz `app/dist/app/browser/index.html`;
- o script pós-build limpa e copia para
  `api/internal/web/dist/browser/`;
- `api/internal/web/dist/browser/index.html` existe;
- nenhum `.env`, banco ou token está no destino.

## 3. Testar e compilar o serviço autocontido

```powershell
cd ../api
go test ./...
go build -o screen-share.exe ./cmd/server
```

O arquivo `screen-share.exe` incorpora o frontend; a pasta
`internal/web/dist/browser` não é necessária ao lado do executável em
runtime.

## 4. Executar somente o serviço

```powershell
if (-not (Test-Path .env)) { Copy-Item .env.example .env }
./screen-share.exe
```

Abrir `http://127.0.0.1:8080`.

Esperado:

1. `/` mostra a tela de gerar link;
2. gerar link usa `/api/v1/links` no mesmo host;
3. o link copiado começa com `http://127.0.0.1:8080/r/`;
4. abrir e recarregar `/r/{id}` mostra a sala;
5. um asset do HTML retorna seu tipo correto;
6. `/api/v1/inexistente` retorna JSON 404, nunca HTML.

## 5. Provar que o binário é autocontido

Copiar apenas o executável para outro diretório (e fornecer um `.env`
válido). Executá-lo nesse diretório e abrir a raiz.

Esperado: a interface aparece. O SQLite é criado conforme configuração,
mas nenhum arquivo Angular externo é necessário.

## 6. Validar falha sem bundle

Em uma cópia de trabalho descartável, limpar
`api/internal/web/dist/browser/` antes de compilar/testar o startup.

Esperado: o serviço recusa iniciar com mensagem clara de frontend
ausente. Restaurar o bundle com `npm run build`.

## Desenvolvimento local

O fluxo existente continua disponível:

```powershell
# terminal 1
cd api
go run ./cmd/server

# terminal 2
cd app
npm start
```

`ng serve` mantém hot reload e proxy para `/api`. O modo incorporado é
o caminho de publicação, não substitui a experiência de edição.
