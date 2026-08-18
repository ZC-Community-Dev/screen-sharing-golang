# Research: 002-embed-frontend

## 1. Fronteira entre Angular e Go

**Decision**: manter o Angular como build independente em `app/` e
copiar somente sua saída compilada para `api/internal/web/dist/browser/`.
O Go incorpora e entrega esses arquivos; não renderiza componentes.

**Rationale**: preserva as árvores e responsabilidades exigidas pela
constituição. O navegador continua executando e renderizando a SPA,
enquanto o Gin mantém regras, autorização, persistência e contratos.

**Alternatives considered**:

- SSR/templates no Go: viola a separação e é desnecessário.
- Servir `app/dist` por caminho relativo em runtime: o binário deixa
  de ser autocontido e quebra ao mudar de diretório.
- Mover fontes Angular para `api/`: viola as árvores canônicas.

## 2. Incorporação dos arquivos no binário

**Decision**: pacote `api/internal/web` com `//go:embed all:dist`,
`fs.Sub` para `dist/browser` e validação de `index.html`.

**Rationale**: `go:embed` é biblioteca padrão, funciona em Windows e
Linux e elimina dependência de arquivos externos no runtime. O padrão
`all:dist` permite manter `.gitkeep` para o pacote Go compilar antes do
primeiro build Angular; a validação explícita impede o servidor de subir
quando só o placeholder estiver incorporado.

**Alternatives considered**:

- Gerador de código/terceiro (`statik`, `packr`): dependência sem ganho.
- `embed.FS` no `main`: acopla entrypoint, assets e roteamento.
- Falhar apenas no `go build` quando dist não existe: mensagem menos
  clara e impede testes unitários do backend antes do build do app.

## 3. Pipeline de build e cópia

**Decision**: `npm run build` executa o build Angular e depois um script
Node ESM em `app/scripts/copy-to-api.mjs`. O script limpa o destino,
valida a origem `dist/app/browser/index.html` e copia recursivamente
para `../api/internal/web/dist/browser/`.

**Rationale**: Node já é requisito do Angular e suas APIs de filesystem
são multiplataforma. Limpar o destino evita que assets hashados antigos
permaneçam incorporados. Validar `index.html` produz erro acionável.

**Alternatives considered**:

- `cp`, `xcopy` ou PowerShell no `package.json`: não é multiplataforma.
- Configurar `outputPath` diretamente dentro de `api/`: mistura saída
  intermediária Angular com o pacote Go e torna limpeza/validação menos
  explícitas.
- Copiar `app/public` sem compilar: não inclui JavaScript, CSS e HTML
  processados necessários à SPA.

## 4. Roteamento estático e fallback SPA

**Decision**: registrar todas as rotas `/api/v1` primeiro e usar
`Engine.NoRoute` para o frontend. Somente GET e HEAD fora de `/api/`
podem receber assets ou `index.html`. Um arquivo existente é servido
com seu tipo de mídia; uma rota sem arquivo recebe `index.html`.
Caminhos `/api` e `/api/*` desconhecidos recebem erro JSON 404.

**Rationale**: o fallback possibilita recarregar `/r/:id` sem capturar
erros da API. Restringir métodos evita transformar POST desconhecido em
HTML. A ordem de registro protege API e WebSocket existentes.

**Alternatives considered**:

- Rota wildcard `/*filepath` antes da API: risco de colisão e conflito
  com rotas Gin.
- Sempre devolver `index.html`: mascara asset ausente e endpoint errado.
- Servidor HTTP separado para estáticos: contradiz o objetivo de um
  processo/origem.

## 5. Cache e segurança

**Decision**: `index.html` usa `Cache-Control: no-cache`; assets com nome
hashado podem usar cache público imutável. O handler limpa o caminho,
impede travessia e serve apenas arquivos presentes no FS embutido.
Nenhum `.env`, salt, banco ou token entra no diretório de build.

**Rationale**: o HTML precisa descobrir o build atual; assets hashados
são seguros para cache longo. Um FS embutido limita a superfície, mas
o caminho ainda deve ser normalizado.

**Alternatives considered**:

- Sem política de cache: funciona, mas piora atualização e desempenho.
- Incluir configuração secreta no environment Angular: segredo seria
  público no bundle e viola a constituição.

## 6. Contratos e testes

**Decision**: não alterar `http-api.yaml`; criar
`contracts/static-web.md` para documentos/arquivos. Testes com
`httptest` cobrem `/`, asset existente, `/r/{id}`, `/api/inexistente`,
método não GET e ausência de `index.html`. O pipeline roda
`npm test`, `npm run build`, `go test ./...` e build do binário.

**Rationale**: nenhum JSON de negócio muda, mas o comportamento HTTP de
documentos merece contrato. Os testes provam a fronteira crítica sem
automatizar permissão real de compartilhamento de tela.

**Alternatives considered**:

- Alterar OpenAPI com documentos HTML: OpenAPI existente descreve a API
  JSON e misturar os dois reduz clareza.
- Somente validação manual: não protege o fallback `/api/`.
