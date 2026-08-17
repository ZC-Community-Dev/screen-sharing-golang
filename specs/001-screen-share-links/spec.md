# Feature Specification: Compartilhamento de Tela por Link

**Feature Branch**: `001-screen-share-links`

**Created**: 2026-08-17

**Status**: Draft

**Input**: User description: "minha aplicação deve ser um sistema para gerenciar links e compartilhar tela, deve ter o visual parecido com o google meet porem sem voz, apenas tela. inclua um botão para gerar um link, esse link deve gerar com um id aleatório com no mínimo 8 caracteres, use base62 com salt para gerar ids aleatórios. use sqlite para guardar os links, e a pessoa que gerar o link tera um token para compartilhar a tela. nesse link ele pode enviar para outras pessoas assistirem a tela."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Gerar e copiar um link de compartilhamento (Priority: P1)

Uma pessoa abre o sistema, vê um botão para gerar um link e o aciona.
O sistema cria um novo espaço de compartilhamento com um identificador
público único e um token secreto de apresentador. A pessoa vê o link
pronto para copiar e enviar a outras pessoas. O token de apresentador
não faz parte do link público.

**Why this priority**: sem o link não existe sessão nem convite; é o
MVP mínimo do produto.

**Independent Test**: gerar um link, copiá-lo e confirmar que o
identificador tem no mínimo 8 caracteres, é único e que o token de
apresentador é exibido apenas para quem gerou o link.

**Acceptance Scenarios**:

1. **Given** a tela inicial sem sessão ativa, **When** a pessoa aciona
   "gerar link", **Then** o sistema cria um identificador público com
   no mínimo 8 caracteres e mostra o link completo para copiar.
2. **Given** um link recém-gerado, **When** a pessoa copia o link,
   **Then** o valor copiado contém apenas o identificador público e
   MUST NOT incluir o token de apresentador.
3. **Given** dois acionamentos seguidos de "gerar link", **When** os
   dois identificadores são comparados, **Then** eles são diferentes e
   cada um tem o próprio token de apresentador.
4. **Given** um link gerado, **When** o serviço é reiniciado, **Then**
   o mesmo identificador continua válido e recuperável.

---

### User Story 2 - Apresentador compartilha a tela (Priority: P1)

Quem gerou o link entra no espaço como apresentador, usando o token
recebido na criação. A pessoa inicia o compartilhamento de tela
(janela, aba ou tela inteira, conforme o navegador permitir). Apenas
quem possui o token consegue iniciar a transmissão. Não há captura
nem transmissão de voz.

**Why this priority**: o valor do produto é mostrar a tela; o token
garante que só o dono do link apresenta.

**Independent Test**: criar um link, entrar como apresentador com o
token, iniciar o compartilhamento e confirmar que a tela aparece no
palco. Tentar iniciar o compartilhamento só com o link público deve
falhar.

**Acceptance Scenarios**:

1. **Given** um link válido e o token de apresentador, **When** a
   pessoa entra no espaço e inicia o compartilhamento de tela, **Then**
   o palco principal passa a exibir a tela compartilhada.
2. **Given** apenas o link público (sem token), **When** alguém tenta
   iniciar o compartilhamento, **Then** o sistema recusa e a pessoa
   permanece como espectadora.
3. **Given** um compartilhamento ativo, **When** a pessoa observa os
   controles, **Then** não há controles de microfone, câmera ou voz
   funcionais; só tela.
4. **Given** um apresentador já transmitindo nesse link, **When**
   outra pessoa tenta apresentar com um token inválido ou ausente,
   **Then** a tentativa é recusada e a transmissão original continua.

---

### User Story 3 - Espectadores assistem pelo link (Priority: P1)

Quem recebe o link abre-o no navegador e entra no espaço como
espectador, sem token. Se o apresentador já estiver compartilhando,
a tela aparece no palco. Se ainda não houver transmissão, a pessoa
vê um estado de espera claro (como uma sala de reunião vazia) e
passa a ver a tela quando o apresentador iniciar.

**Why this priority**: o convite só tem valor se outras pessoas
conseguirem assistir.

**Independent Test**: abrir o link público em outro navegador ou
perfil e verificar que a tela compartilhada (ou o estado de espera)
aparece, sem opção de apresentar.

**Acceptance Scenarios**:

1. **Given** um compartilhamento ativo, **When** um espectador abre o
   link público, **Then** a tela do apresentador aparece no palco
   sem pedir token.
2. **Given** um link válido sem transmissão ativa, **When** um
   espectador entra, **Then** vê um estado de espera e, quando o
   apresentador inicia, passa a ver a tela sem recarregar o convite.
3. **Given** vários espectadores no mesmo link, **When** o
   apresentador está compartilhando, **Then** todos vêem a mesma
   tela e nenhum consegue iniciar uma segunda transmissão.
4. **Given** um identificador inexistente ou malformado, **When**
   alguém abre o endereço, **Then** vê uma mensagem de link inválido
   e não entra em um espaço vazio genérico.

---

### User Story 4 - Sala com visual de reunião, só tela (Priority: P2)

Quem entra no espaço (apresentador ou espectador) vê um layout
inspirado em uma reunião por vídeo: palco central grande, barra de
controles na parte inferior, indicação de quantas pessoas estão na
sala, e ação para copiar o link. Não há voz, câmera dos
participantes nem chat. O palco mostra só a tela compartilhada ou
o estado de espera.

**Why this priority**: o usuário pediu explicitamente a experiência
visual de uma reunião, restrita a tela; não bloqueia o fluxo P1 de
gerar, apresentar e assistir.

**Independent Test**: entrar como apresentador e como espectador e
confirmar palco central, barra inferior, contagem de pessoas, cópia
de link, ausência de voz/câmera/chat.

**Acceptance Scenarios**:

1. **Given** um espaço aberto, **When** a pessoa olha a sala, **Then**
   o palco ocupa a área principal e os controles ficam em uma barra
   inferior persistente.
2. **Given** apresentador e pelo menos um espectador, **When**
   qualquer um olha a indicação de presença, **Then** o número de
   pessoas na sala está visível e atualizado.
3. **Given** a barra de controles, **When** a pessoa a inspeciona,
   **Then** existe ação para copiar o link público e não existem
   ações de microfone, câmera ou chat.
4. **Given** transmissão ativa, **When** um espectador assiste,
   **Then** a tela compartilhada preenche o palco (letterbox se a
   proporção for diferente), sem sobrepor áudio ou vídeo de câmera.

---

### User Story 5 - Encerrar o compartilhamento (Priority: P3)

O apresentador para de compartilhar a tela. Os espectadores voltam
ao estado de espera no mesmo link. O apresentador pode iniciar de
novo com o mesmo token. Encerrar a transmissão não apaga o link.

**Why this priority**: ciclo completo da sessão; o MVP já entrega
valor com gerar + apresentar + assistir.

**Independent Test**: iniciar, parar e reiniciar o compartilhamento
no mesmo link; espectadores devem ir para espera e voltar a ver a
tela.

**Acceptance Scenarios**:

1. **Given** uma transmissão ativa, **When** o apresentador encerra o
   compartilhamento, **Then** o palco de todos volta ao estado de
   espera e o link continua válido.
2. **Given** um link cujo compartilhamento foi encerrado, **When** o
   apresentador inicia de novo com o token correto, **Then** os
   espectadores já presentes passam a ver a nova tela.
3. **Given** o apresentador desconecta de forma inesperada (fecha a
   aba), **When** os espectadores permanecem no link, **Then** o palco
   volta à espera em no máximo alguns segundos e ninguém assume o
   papel de apresentador.

---

### Edge Cases

- Identificador com menos de 8 caracteres, caracteres fora do alfabeto
  permitido, ou vazio: tratar como link inválido.
- Colisão de identificador na geração: não reutilizar; gerar outro
  até obter um identificador único.
- Token ausente, truncado ou de outro link: recusar apresentação;
  permitir apenas assistência se o link público for válido.
- Espectador entra com o link público enquanto o apresentador ainda
  não escolheu o que compartilhar: estado de espera, sem erro.
- Dois apresentadores com o mesmo token em dispositivos diferentes:
  apenas uma transmissão ativa por link; a segunda tentativa é
  recusada ou substitui de forma explícita a primeira — nesta versão,
  a segunda MUST ser recusada enquanto a primeira estiver ativa.
- Perda de rede do apresentador: espectadores vão à espera; ao
  reconectar com o token, o apresentador pode retomar.
- Perda de rede do espectador: ao reconectar no mesmo link, volta a
  assistir ou à espera, conforme o estado atual.
- Serviço reiniciado no meio de uma transmissão: os registros de
  link e token persistem; a transmissão ao vivo precisa ser
  reestabelecida pelo apresentador.
- Muitos espectadores no mesmo link: todos assistem; ninguém apresenta
  sem token.
- Pessoa cola o token no lugar do identificador público (ou o
  contrário): mensagem de link inválido ou acesso apenas como
  espectador, nunca promoção acidental a apresentador.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O sistema MUST permitir gerar um novo link de
  compartilhamento a partir de um botão visível na tela inicial,
  sem exigir conta ou login.
- **FR-002**: Cada link MUST ter um identificador público único com
  no mínimo 8 caracteres, composto somente pelo alfabeto de 62
  símbolos (dígitos 0-9, letras maiúsculas A-Z e minúsculas a-z).
- **FR-003**: Identificadores MUST ser imprevisíveis: MUST NOT ser
  sequenciais nem derivados de forma reversível da ordem de criação.
  A geração MUST usar um salt secreto do servidor para que o
  identificador público não possa ser calculado por terceiros.
- **FR-004**: Ao gerar o link, o sistema MUST emitir um token de
  apresentador vinculado exclusivamente àquele link. O token MUST
  ser mostrado a quem gerou o link e MUST NOT aparecer no endereço
  público enviado aos espectadores.
- **FR-005**: O sistema MUST persistir cada link (identificador,
  token de apresentador, data de criação e estado) de forma que
  sobreviva a reinício do serviço.
- **FR-006**: Quem possui o token de apresentador MUST poder iniciar
  e encerrar o compartilhamento de tela naquele link.
- **FR-007**: Quem acessa apenas o link público MUST poder assistir
  e MUST NOT poder iniciar compartilhamento.
- **FR-008**: O sistema MUST aceitar vários espectadores
  simultâneos no mesmo link e MUST permitir no máximo um
  apresentador ativo por link.
- **FR-009**: O sistema MUST NOT capturar, transmitir nem reproduzir
  voz, microfone ou câmera dos participantes.
- **FR-010**: Enquanto não houver tela compartilhada, apresentador e
  espectadores MUST ver um estado de espera compreensível no palco.
- **FR-011**: Quando o apresentador inicia ou encerra a
  transmissão, os espectadores já presentes MUST atualizar o palco
  sem precisar gerar um novo link.
- **FR-012**: Acesso a identificador desconhecido ou malformado MUST
  resultar em mensagem de link inválido, sem criar um espaço novo.
- **FR-013**: A sala MUST apresentar palco central para a tela,
  barra de controles inferior, indicação de quantas pessoas estão
  presentes e ação para copiar o link público, em um visual
  reconhecível como sala de reunião (referência: reunião por vídeo
  em tela cheia), sem controles de voz, câmera ou chat.
- **FR-014**: O apresentador MUST poder copiar o link público em um
  único acionamento a partir da sala.
- **FR-015**: Encerrar o compartilhamento MUST manter o link e o
  token válidos para uma nova transmissão posterior.
- **FR-016**: Tokens e salts MUST NOT ser escritos em registros de
  diagnóstico, endereços públicos ou respostas destinadas a
  espectadores.

### Key Entities

- **Link de compartilhamento**: convite público para um espaço de
  tela. Atributos: identificador público (≥8 caracteres, alfabeto de
  62 símbolos), data de criação, estado (aguardando / transmitindo /
  inválido).
- **Token de apresentador**: credencial secreta emitida na criação
  do link; autoriza iniciar e encerrar a transmissão daquele link;
  não é o mesmo valor que o identificador público.
- **Sessão de visualização**: presença temporária de uma pessoa no
  espaço (apresentador ou espectador); não exige cadastro.
- **Transmissão de tela**: o conteúdo visual enviado pelo
  apresentador ao palco; um único fluxo ativo por link; sem áudio.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Uma pessoa sem cadastro consegue gerar um link e
  copiá-lo em menos de 30 segundos a partir da tela inicial.
- **SC-002**: 100% dos identificadores gerados têm no mínimo 8
  caracteres, usam somente o alfabeto de 62 símbolos e são únicos
  entre todos os links persistidos.
- **SC-003**: Em 95% das tentativas, um espectador que abre um link
  com transmissão já ativa vê a tela no palco em menos de 5 segundos
  (rede local ou equivalente estável).
- **SC-004**: Em 100% dos testes, um visitante com apenas o link
  público não consegue iniciar o compartilhamento.
- **SC-005**: Pelo menos 10 espectadores simultâneos no mesmo link
  vêem a tela do apresentador sem precisar de voz ou cadastro.
- **SC-006**: Após reinício do serviço, um link gerado antes do
  reinício continua abrível pelo mesmo identificador público.
- **SC-007**: 90% das pessoas reconhecem a sala como um espaço de
  reunião em tela (palco + barra inferior + pessoas + copiar link)
  e confirmam a ausência de voz na primeira visita.
- **SC-008**: Ao encerrar a transmissão, 100% dos espectadores
  presentes voltam ao estado de espera em menos de 5 segundos, e o
  mesmo link permanece utilizável.

## Assumptions

- Não há contas, login nem lista histórica de “meus links”. Gerenciar
  o link significa criar, copiar, apresentar, assistir e encerrar a
  transmissão daquele convite.
- O salt secreto de geração de identificadores é configuração de
  servidor, não escolhido pelo usuário a cada link.
- O alfabeto de 62 símbolos e o salt na geração de IDs são restrição
  do stakeholder; o planejamento MUST honrá-los (Base62 + salt).
- A persistência dos links MUST usar um banco embarcado em arquivo
  local, conforme pedido do stakeholder (SQLite); o planejamento
  MUST tratar isso como restrição vinculante, não como opção.
- Navegadores modernos com permissão de captura de tela são o
  ambiente suportado; aplicativo nativo móvel fica fora desta
  versão.
- Chat, gravação, controle remoto da tela, legendas, fundo
  virtual, câmera e áudio estão fora de escopo.
- Qualidade visual segue o que o navegador entregar na captura;
  não há ajuste fino de resolução nesta versão.
- O token de apresentador fica disponível para quem gerou o link
  naquela sessão do navegador (exibido na criação e reutilizado
  enquanto a aba de apresentador permanecer); não há recuperação
  por e-mail.
- Um link não expira por tempo nesta versão; permanece válido até
  uma versão futura definir revogação explícita.
- A constituição do projeto (cliente/servidor separados, contratos
  HTTP, testes primeiro) aplica-se a esta feature; detalhes de
  stack ficam para o plano, não para os critérios de sucesso.
