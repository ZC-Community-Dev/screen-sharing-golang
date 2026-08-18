# Research: 001-screen-share-links

## 1. Transporte da tela compartilhada

**Decision**: WebRTC (somente trilha de vídeo) em malha apresentador → cada
espectador. Sinalização e presença via WebSocket no Gin. `getDisplayMedia`
com `audio: false`.

**Rationale**: SC-003 (tela visível em <5s) e SC-005 (≥10 espectadores) exigem
um fluxo de mídia contínuo. WebRTC é o caminho nativo do navegador para
captura de tela e entrega em tempo quase real, sem servidor de mídia.

**Alternatives considered**:

- JPEG/canvas por HTTP ou WebSocket: mais simples no servidor, mas falha em
  fluidez, escala e uso de CPU; risco alto de não cumprir SC-003/SC-005.
- SFU (Pion) no backend: melhor para dezenas de espectadores, mas adiciona
  um serviço de mídia e contradiz YAGNI para o mínimo de 10 espectadores.
- Gravação MediaRecorder + chunks: latência pior e buffer mais complexo
  que um `RTCPeerConnection` por espectador.

## 2. Plano de controle vs plano de mídia

**Decision**: CRUD de links, claim de apresentador e start/stop permanecem
HTTP JSON no Gin. Eventos ao vivo (estado da sala, presença, SDP/ICE)
usam WebSocket auxiliar. A mídia não passa pelo JSON.

**Rationale**: a constituição exige contratos HTTP JSON para funcionalidade
de negócio e autoriza WebSocket quando o spec exige atualização sem reload
(FR-011, FR-013, SC-008). WebRTC é canal de mídia justificado em
Complexity Tracking; não substitui Gin.

**Alternatives considered**:

- Sinalização só por POST HTTP: possível, mas ICE trickle e presença
  ficam lentos ou exigem polling; piora SC-008.
- Tudo no WebSocket: foge do princípio II (contrato HTTP JSON testável
  para criar/abrir link e autorizar apresentador).

## 3. Driver SQLite

**Decision**: `modernc.org/sqlite` (puro Go) via `database/sql`. Arquivo
em `api/data/links.db` (gitignored). Na suíte de testes, arquivo temporário
por teste.

**Rationale**: o ambiente é Windows; CGO (`mattn/go-sqlite3`) complica o
build. Driver puro Go cumpre a constituição (SQLite como fonte de verdade)
sem toolchain extra.

**Alternatives considered**:

- `mattn/go-sqlite3`: maduro, mas exige CGO/gcc.
- SQLite só em memória como fonte de verdade: proibido pela constituição
  e quebra SC-006 (sobrevida a reinício).

## 4. Geração de IDs Base62 com sal

**Decision**: 16 bytes de `crypto/rand` → HMAC-SHA256(sal, bytes) →
codificação Base62 (`0-9A-Za-z`) → recortar 10 caracteres (≥8). Em
colisão no SQLite, repetir. Sal obrigatório em `LINK_ID_SALT` no
processo do servidor; falha ao arrancar se vazio.

**Rationale**: HMAC com sal impede IDs sequenciais ou previsíveis a
partir da ordem de criação (FR-003). 10 caracteres no alfabeto de 62
reduz colisão sem alongar o link. O sal nunca vai ao cliente nem ao log
(FR-016, princípio V).

**Alternatives considered**:

- UUID/nanoid sem sal: proibidos pela constituição.
- Contador + Base62: previsível e reversível.
- Só `rand` Base62 sem HMAC: imprevisível, mas o stakeholder exigiu sal.

## 5. Token de apresentador

**Decision**: 32 bytes aleatórios, encode Base62, devolver em claro só
na resposta de criação. Persistir `SHA-256(token)` no SQLite. O Angular
guarda o token em `sessionStorage` da aba que gerou o link. Claim de
apresentador compara hash. Respostas de espectador nunca incluem token.

**Rationale**: FR-004/FR-016 pedem token secreto e fora do URL público.
Hash no disco reduz vazamento se o arquivo SQLite for copiado. Sem contas,
`sessionStorage` é o default da spec (sem recuperação por e-mail).

**Alternatives considered**:

- Token no query string (`?token=`): vaza em histórico, Referer e logs
  de proxy.
- Guardar token em claro no SQLite: desnecessário e pior em vazamento.
- Cookie HttpOnly compartilhado: não funciona para o apresentador abrir
  o mesmo link noutro perfil/dispositivo com o token copiado à mão.

## 6. Presença e estado após reinício

**Decision**: presença (sessões) é só memória no processo. Ao subir, todo
link persistido com `state = sharing` volta para `waiting`. Transmissão
ao vivo não sobrevive a restart (alinhado à spec).

**Rationale**: persistir quem está na sala exigiria heartbeat e limpeza;
YAGNI. SC-006 exige só que o *link* continue abrível, não a mídia.

**Alternatives considered**:

- Tabela de sessões no SQLite: complexidade sem ganho se o processo
  morreu (WebRTC já caiu).
- Manter `sharing` após restart: espectadores veriam “transmitindo”
  sem tela.

## 7. Árvores do repositório

**Decision**: backend em `api/` (Go/Gin + SQLite) e frontend em `app/`
(Angular 21 + Tailwind 4). Esses caminhos são obrigatórios.

**Rationale**: constituição 1.3.0 fixa `api/` e `app/` como árvores
canônicas. Renomear, fundir ou inverter exige emenda. O repositório
já está nesse layout.

**Alternatives considered**:

- `backend/` e `frontend/`: eram exemplo na v1.2.0; a v1.3.0
  proíbe esse rename sem emenda.
- Monólito que serve o Angular pelo Gin em dev: mistura de artefatos
  e atrasa o contrato HTTP testável.

## 8. Testes

**Decision**: backend com `testing` + `httptest` + Gin em modo teste.
Frontend com Vitest (já no `app/`). Contratos HTTP cobrem cada path
público. Integração sobe o servidor Gin real contra SQLite temporário.
Nenhum teste de mídia WebRTC ponta a ponta nesta versão; o contrato
de sinalização e start/stop é testado.

**Rationale**: princípios III e IV. WebRTC real depende de hardware e
permissão de tela; automatizar isso agora viola YAGNI.

**Alternatives considered**:

- Playwright com fake media: útil depois; não bloqueia o plano.
- Sem testes de contrato: viola a constituição.

## 9. Configuração do Angular vs segredos da API

**Decision**: as principais opções **públicas** do cliente ficam em
`app/src/environments/environment.ts` (produção) e
`environment.development.ts` (`ng serve`, fileReplacements). Campos:
`production`, `apiBaseUrl`, `roomPathPrefix`, `stunUrls`, `appOrigin`.
Helpers em `app/src/app/config.ts` montam HTTP, WebSocket e ICE.
Segredos do servidor (`LINK_ID_SALT`, `LINKS_DB_PATH`, `PORT`)
permanecem só em `api/.env`.

**Rationale**: o utilizador gerou os environments no `app/`. File
replacement do Angular é compile-time, não dotenv. A constituição
proíbe expor o sal ao cliente; environments Angular MUST NOT
duplicar `LINK_ID_SALT` nem o token de apresentador. Em dev,
`apiBaseUrl: '/api/v1'` usa o proxy já existente.

**Alternatives considered**:

- Dotenv no Angular (`@ngx-env`, `app/.env`): mistura com o
  `api/.env` e aumenta o risco de copiar o sal para o bundle.
- Hardcode `/api/v1` e STUN nos serviços: funciona, mas impede
  apontar o build de produção para outro host sem editar código.
- URL absoluta `http://127.0.0.1:8080/api/v1` no development:
  bypassa o proxy e depende de CORS/WS origin; o proxy same-origin
  é o default mais simples.
