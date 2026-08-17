# Quickstart: 001-screen-share-links

Guia de validação ponta a ponta. Não substitui `tasks.md` nem a suíte
de testes. Contratos: [http-api.yaml](./contracts/http-api.yaml),
[room-events.md](./contracts/room-events.md). Modelo:
[data-model.md](./data-model.md).

## Prerequisites

- Go 1.25+
- Node.js com npm (Angular 21 no `app/`)
- Arquivo `api/.env` com `LINK_ID_SALT` não vazio (copie `api/.env.example`).
  Variáveis já exportadas no shell prevalecem sobre o arquivo.
- Dois perfis de navegador (apresentador e espectador)

## Setup

```powershell
# backend — lê api/.env automaticamente (não precisa exportar o sal)
cd api
if (-not (Test-Path .env)) { Copy-Item .env.example .env }
go test ./...
go run ./cmd/server

# frontend (outro terminal)
cd app
npm test
npm start
```

API em `http://127.0.0.1:8080`. UI em `http://127.0.0.1:4200` com proxy
para `/api`.

## Validation scenarios

### 1. Gerar e copiar link (P1)

1. Abrir a tela inicial.
2. Acionar **Gerar link**.
3. Confirmar id com ≥8 caracteres Base62, URL pública sem token, e o
   token de apresentador visível na tela.
4. Copiar o link: a área de transferência MUST ser só a URL pública.
5. Reiniciar o processo `api` e abrir `GET /api/v1/links/{id}` — 200.

Esperado: SC-001, SC-002, SC-006.

### 2. Apresentar com token (P1)

1. Na aba que gerou o link, entrar na sala (token em `sessionStorage`).
2. Iniciar compartilhamento de tela (sem áudio).
3. Palco do apresentador mostra a captura.
4. Noutro perfil, abrir só a URL pública e tentar iniciar share — recusa.

Esperado: SC-004; controles sem microfone/câmera.

### 3. Assistir pelo link (P1)

1. Com partilha ativa, abrir a URL pública noutro perfil.
2. A tela aparece no palco em poucos segundos, sem pedir token.
3. Abrir um id curto ou com caracteres inválidos — mensagem de link
   inválido, sem sala vazia.

Esperado: SC-003; FR-012.

### 4. Sala estilo reunião (P2)

1. Conferir palco central, barra inferior, contagem de pessoas e copiar
   link.
2. Confirmar ausência de chat, microfone e câmera.

Esperado: SC-007.

### 5. Encerrar e retomar (P3)

1. Apresentador para a partilha — espectadores voltam à espera (<5s).
2. Apresentador inicia de novo com o mesmo token — a tela volta.
3. Fechar a aba do apresentador — espera automática; ninguém vira host.

Esperado: SC-008.

## Out of scope for this guide

Implementação de handlers, migrations e suítes completas. Isso fica em
`/speckit-tasks` e `/speckit-implement`.
