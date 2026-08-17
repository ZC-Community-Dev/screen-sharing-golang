<!--
Sync Impact Report
- Version change: 1.0.0 → 1.1.0
- Modified principles: none (títulos e regras I–V inalterados)
- Added sections: none
- Removed sections: none
- Material expansions:
  - Stack Tecnológica: Tailwind CSS passa a ser a camada canônica de
    estilo do Angular; outros frameworks CSS exigem emenda
  - Fluxo de Desenvolvimento: reviews MUST checar Tailwind CSS na stack
- Follow-up TODOs: none
-->

# Screen Share Constitution

## Core Principles

### I. Separação Cliente-Servidor
O sistema MUST ser composto por dois artefatos distintos: um backend em
Go com Gin e um frontend em Angular. A comunicação entre eles MUST
ocorrer apenas por APIs HTTP documentadas. Lógica de negócio, autorização
e persistência MUST residir no backend. O frontend MUST limitar-se a
apresentação, estado de UI e consumo dos contratos públicos. Código Go
MUST NOT ser embutido no app Angular, e o backend MUST NOT renderizar
UI Angular.

**Rationale**: fronteira clara reduz acoplamento, permite evoluir cada
lado de forma independente e torna contratos testáveis.

### II. Contrato HTTP JSON
Toda funcionalidade exposta ao cliente MUST ser um endpoint HTTP JSON
servido pelo Gin. Cada endpoint MUST declarar método, caminho, corpo de
entrada, corpo de saída e códigos de erro. O Angular MUST consumir
somente esses contratos públicos. Mudança incompatível de contrato MUST
incrementar a versão MAJOR da API e registrar nota de migração antes do
merge.

**Rationale**: contratos explícitos evitam drift entre Gin e Angular e
tornam quebras detectáveis em review e teste.

### III. Testes Primeiro (NÃO NEGOCIÁVEL)
Nenhuma feature MUST ser implementada sem testes escritos antes do
código de produção. O ciclo obrigatório é: escrever testes → obter
aprovação do comportamento esperado → confirmar falha → implementar →
refatorar. O backend MUST ter testes Go para handlers, middleware e
regras de negócio. O frontend MUST ter testes Angular para componentes
e serviços que tocam a API. Endpoint novo ou contrato alterado MUST
incluir teste de contrato.

**Rationale**: TDD fixa o comportamento antes da implementação e impede
regressão na fronteira Go/Angular.

### IV. Testes de Integração na Fronteira
Os seguintes casos MUST ter testes de integração: endpoint Gin novo,
alteração de contrato JSON, autenticação/autorização, e o fluxo
Angular → HTTP → Gin. Testes de integração MUST exercitar o servidor
HTTP real (ou equivalente de teste do Gin) e MUST falhar se o contrato
publicado divergir do cliente.

**Rationale**: a falha mais cara deste projeto é dessincronia entre
frontend e backend; a fronteira HTTP é o ponto de prova.

### V. Simplicidade e Observabilidade
O desenho MUST começar pelo caminho mais simples que atenda o spec.
Dependência, camada ou abstração extra MUST ser justificada por
requisito existente, não por hipótese futura. O backend MUST emitir
logs estruturados (nível, correlação de request, erro). Segredos MUST
NOT aparecer em logs, commits ou respostas de API.

**Rationale**: YAGNI reduz custo de mudança; logs estruturados tornam
falhas de share e API diagnosticáveis sem debugger ad hoc.

## Stack Tecnológica

O backend MUST ser escrito em Go e MUST usar Gin como único framework
HTTP. Outros roteadores ou frameworks Go (Echo, Fiber, Chi, net/http
puro como API pública) MUST NOT ser adotados sem emenda a esta
constituição.

O frontend MUST ser uma aplicação Angular. Outros frameworks de UI
(React, Vue, Svelte, templates server-side) MUST NOT substituir o
Angular sem emenda a esta constituição.

A estilização do frontend MUST usar Tailwind CSS como camada canônica
de CSS. Componentes Angular MUST aplicar utilitários Tailwind (e
tokens/tema do projeto) em vez de um framework CSS concorrente.
Bootstrap, Foundation, CSS-in-JS como sistema principal, ou folhas
globais ad hoc que substituam o Tailwind MUST NOT ser adotados sem
emenda a esta constituição. Estilos pontuais de componente MAY existir
quando um utilitário Tailwind não cobrir o caso, e MUST permanecer
excepcionais e justificados no spec ou no plano.

O repositório MUST manter backend e frontend em árvores separadas
(por exemplo `backend/` e `frontend/`). Dependências de cada lado
MUST ser geridas pelas ferramentas nativas (Go modules e Angular CLI /
npm ou equivalente do workspace Angular).

Bibliotecas auxiliares (validação, CORS, WebSocket, persistência) MAY
ser adicionadas quando um spec o exigir. Elas MUST NOT substituir Gin,
Angular nem Tailwind CSS como stack canônica.

## Fluxo de Desenvolvimento

Toda feature MUST seguir o fluxo Spec Kit: especificação → plano →
tarefas → implementação. Código de produção MUST NOT ser o primeiro
artefato de uma feature.

Pull requests e reviews MUST verificar conformidade com esta
constituição, em especial: stack Go+Gin / Angular / Tailwind CSS,
contratos HTTP documentados, testes escritos antes da
implementação, e ausência de segredos.

Complexidade adicional (novo serviço, novo canal além de HTTP JSON,
novo framework) MUST ser justificada no plano da feature e, se
contrariar um princípio, MUST passar pelo processo de emenda antes
da implementação.

## Governance

Esta constituição prevalece sobre convenções locais, snippets e
preferências ad hoc. Em caso de conflito, o texto aqui MUST ser
seguido até ser emendado.

Emendas MUST: (1) alterar este arquivo; (2) atualizar a versão
segundo SemVer; (3) registrar o impacto no Sync Impact Report no topo
do arquivo; (4) definir `Last Amended` com a data ISO do dia da
mudança. Remoção ou redefinição incompatível de princípio é MAJOR.
Princípio ou seção nova, ou expansão material de orientação, é MINOR.
Ajuste de redação sem mudar o significado é PATCH.

Reviews de conformidade MUST ocorrer a cada feature (specify/plan/
tasks/implement) e a cada PR. Desvio temporário MUST NOT ser
silencioso: ou a feature é ajustada, ou a constituição é emendada.

**Version**: 1.1.0 | **Ratified**: 2026-08-17 | **Last Amended**: 2026-08-17
