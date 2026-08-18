# Feature Specification: Interface no mesmo endereço do serviço

**Feature Branch**: `002-embed-frontend`

**Created**: 2026-08-17

**Status**: Draft

**Input**: User description: "eu preciso o gin do go lang carregue o frontend, então ao buildar o app ele deve mover a pasta pública para a api, e nele deve ter um enbeding no golang que expoem a tela entendeu?"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Abrir um único endereço e ver o produto (Priority: P1)

Quem publica ou usa o sistema sobe **um** serviço. Ao abrir o
endereço desse serviço no navegador, vê a tela inicial do produto
(gerar link, entrar na sala) — sem precisar ligar um segundo
programa só para a interface.

**Why this priority**: hoje a interface e o serviço de links são
dois processos; o pedido é um único ponto de entrada que já
mostra as telas.

**Independent Test**: publicar o sistema, abrir só o endereço do
serviço e confirmar que a tela inicial aparece e o botão de gerar
link funciona.

**Acceptance Scenarios**:

1. **Given** o serviço publicado em um endereço, **When** uma
   pessoa abre esse endereço na raiz, **Then** vê a tela inicial
   do compartilhamento (gerar link), não uma página vazia nem um
   erro de “não encontrado”.
2. **Given** a tela inicial servida por esse endereço, **When** a
   pessoa gera um link, **Then** o fluxo existente de criar e
   copiar o convite funciona como hoje.
3. **Given** o serviço publicado, **When** a pessoa não inicia
   nenhum outro programa de interface, **Then** ainda consegue
   usar o produto só com aquele endereço.

---

### User Story 2 - Convites e salas no mesmo endereço (Priority: P1)

O link público que a pessoa copia aponta para o **mesmo**
endereço do serviço. Quem recebe o convite abre a sala nesse
host. Recarregar a página da sala (ou um endereço de sala
inválido) continua mostrando a interface certa, não um erro
cru de arquivo em falta.

**Why this priority**: o valor do produto é o convite; se a sala
não abrir no mesmo host, o empacotamento único falha.

**Independent Test**: gerar um link, copiá-lo, abrir `/r/{id}`
nesse host (e recarregar). Confirmar a sala ou a mensagem de
link inválido.

**Acceptance Scenarios**:

1. **Given** um link gerado no serviço publicado, **When** o
   convite é aberto noutro perfil no mesmo host, **Then** a
   pessoa entra na sala (espera ou tela) sem um segundo servidor
   de interface.
2. **Given** uma sala aberta, **When** a pessoa recarrega o
   endereço da sala, **Then** volta a ver a sala (ou inválido),
   não uma falha de “página não existe”.
3. **Given** um identificador inválido nesse host, **When**
   alguém abre o endereço da sala, **Then** vê a mensagem de
   link inválido já existente.

---

### User Story 3 - Publicar interface junto com o serviço (Priority: P2)

Quem prepara uma versão para publicar executa o build da
interface. Os arquivos públicos resultantes passam a fazer parte
do pacote do serviço. Ao arrancar o serviço, essas telas já
estão **dentro** dele (não dependem de uma pasta solta no
computador de quem abre o navegador).

**Why this priority**: sem este passo, o endereço único não é
reproduzível noutro computador.

**Independent Test**: build da interface → arquivos no pacote do
serviço → arrancar só o serviço noutro diretório e abrir o
endereço: a tela aparece.

**Acceptance Scenarios**:

1. **Given** um build da interface concluído, **When** o pacote
   do serviço é inspecionado, **Then** os arquivos públicos da
   interface estão lá (não só o programa do serviço).
2. **Given** esse pacote, **When** o serviço sobe sem um
   servidor de interface à parte, **Then** o endereço mostra as
   telas.
3. **Given** o pacote copiado para outra máquina com o mesmo
   tipo de publicação, **When** só o serviço é iniciado, **Then**
   a interface continua disponível (os arquivos viajam com o
   serviço).

---

### User Story 4 - Serviço de links continua distinto das telas (Priority: P2)

Os pedidos de criar link, entrar na sala e eventos ao vivo
continuam no caminho de serviço já conhecido. Pedir um arquivo
da interface MUST NOT “engolir” esses caminhos. A interface no
navegador continua a falar com o serviço pelos contratos
públicos; o servidor **entrega** as telas já construídas, não as
monta de novo a cada clique.

**Why this priority**: misturar tela e API no mesmo endereço não
pode quebrar o compartilhamento nem os testes de contrato.

**Independent Test**: com o serviço publicado, gerar link e
assistir pelo convite; confirmar que criar/abrir link e a sala
ao vivo ainda funcionam.

**Acceptance Scenarios**:

1. **Given** o serviço publicado, **When** alguém cria ou abre
   um link pelos caminhos de serviço, **Then** as respostas
   continuam as mesmas do produto atual.
2. **Given** um caminho de serviço (criar link, eventos da
   sala), **When** o navegador pede a interface, **Then** esse
   caminho de serviço não é substituído por uma tela HTML.
3. **Given** a interface servida no mesmo endereço, **When** a
   pessoa usa gerar / apresentar / assistir, **Then** o
   comportamento de negócio não muda.

---

### Edge Cases

- Pedido a um arquivo estático que existe (script, estilo,
  ícone): devolve esse arquivo, não a tela inicial.
- Pedido a um caminho de sala ou outra rota da interface que
  não é arquivo: devolve a tela da aplicação para o navegador
  resolver (recarregar não quebra).
- Pedido a um caminho reservado do serviço de links: nunca
  devolve a tela no lugar da resposta de serviço.
- Pacote publicado sem os arquivos da interface: o serviço
  MUST falhar ao arrancar com erro claro, não subir “mudo”
  com páginas vazias.
- Desenvolvimento local com recarga rápida da interface MAY
  continuar em dois processos; o modo **publicado** é um
  processo só.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O serviço publicado MUST entregar a tela inicial
  do produto no endereço raiz, sem um segundo programa de
  interface.
- **FR-002**: O mesmo endereço MUST entregar as telas de sala e
  de link inválido quando a pessoa abre ou recarrega esses
  caminhos.
- **FR-003**: O build da interface MUST copiar os arquivos
  públicos resultantes para o pacote do serviço, de forma que
  publiquem juntos.
- **FR-004**: Os arquivos da interface MUST viajar **dentro**
  do serviço publicado (embutidos no artefato), não exigir uma
  pasta externa no computador de quem só abre o navegador.
- **FR-005**: Ao arrancar, o serviço MUST expor essas telas no
  mesmo host em que responde aos pedidos de link.
- **FR-006**: Caminhos do serviço de links (criar, abrir, sessões,
  start/stop, eventos ao vivo) MUST continuar a responder como
  hoje e MUST NOT ser substituídos pela tela inicial.
- **FR-007**: A interface no navegador MUST continuar a usar só
  os contratos públicos de link e sala; o empacotamento MUST NOT
  exigir contas novas nem mudar regras de token.
- **FR-008**: Segredos do servidor (salt, token em claro) MUST
  NOT ser copiados para os arquivos públicos da interface.
- **FR-009**: Se os arquivos da interface estiverem ausentes no
  pacote, o serviço MUST recusar arrancar e MUST explicar que a
  interface não foi publicada com ele.
- **FR-010**: Arquivos estáticos nomeados (scripts, estilos,
  ícones) MUST ser servidos como arquivo; rotas da interface
  sem arquivo MUST cair na tela da aplicação.

### Key Entities

- **Pacote publicado**: o serviço de links mais os arquivos
  públicos da interface, prontos para um único arranque.
- **Endereço público**: um host/porta onde a pessoa abre as
  telas e também o serviço de convites.
- **Arquivos da interface**: o resultado do build da tela
  (incluindo o que era estático/público no cliente); não são
  links nem tokens.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Uma pessoa sobe só o serviço publicado e, em
  menos de 15 segundos, vê a tela de gerar link ao abrir o
  endereço (sem segundo programa de interface).
- **SC-002**: 100% dos convites copiados nesse modo abrem a
  sala no mesmo host (espera ou tela), noutro perfil.
- **SC-003**: Em 100% dos testes, recarregar o endereço da
  sala mostra de novo a sala ou a mensagem de link inválido,
  nunca uma falha vazia de “não encontrado” para rotas da
  interface.
- **SC-004**: 100% dos fluxos já existentes (gerar, apresentar,
  assistir, encerrar) completam nesse endereço único.
- **SC-005**: Copiar o pacote publicado para outro diretório
  ou máquina equivalente e arrancar só o serviço ainda mostra
  as telas (os arquivos da interface foram com o pacote).
- **SC-006**: Pedidos de serviço de links continuam a
  funcionar; nenhum deles devolve a tela inicial no lugar da
  resposta de negócio.

## Assumptions

- Esta feature **não** muda o produto de compartilhar tela: só
  o modo de **publicar e abrir** a interface.
- O servidor **entrega** telas já construídas no navegador; não
  monta a interface a cada clique (não é renderização no
  servidor). Isso respeita a separação cliente/servidor: a
  lógica de link continua no serviço; o navegador continua a
  desenhar a sala.
- Restrição do stakeholder para o plano: o build do cliente
  MUST colocar os arquivos públicos no pacote do serviço
  (`api/`); o serviço MUST embuti-los no artefato e expô-los
  no mesmo processo HTTP. Detalhes de ferramenta ficam no
  plano, não nos critérios de sucesso.
- Desenvolvimento local MAY manter dois processos (interface
  com recarga rápida + serviço) para quem está a editar telas.
  O modo publicado é o que esta spec exige.
- Prefixo dos pedidos de link permanece o já usado pelo
  produto (`/api/...`); a interface usa o resto do host
  (raiz, sala, estáticos).
- Sem CDN, sem hospedagem da interface noutro domínio nesta
  versão. `appOrigin` vazio (origem atual) continua correto
  porque tudo é o mesmo endereço.
- A constituição (contratos HTTP, testes primeiro, segredos
  só no serviço, árvores `api/` e `app/`) aplica-se; o plano
  MUST justificar servir arquivos estáticos no mesmo processo
  como entrega do cliente, não como UI gerada no servidor.
