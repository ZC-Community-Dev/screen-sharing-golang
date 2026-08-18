# Feature Specification: Transmissão de tela mediada pelo servidor

**Feature Branch**: `003-server-relay-screen`

**Created**: 2026-08-17

**Status**: Draft

**Input**: User description: "agora preciso que vc reformule o link,
fazendo a conexão passar pelo servidor, n use mais p2p"

## Clarifications

### Session 2026-08-17

- Q: A opção “UDP” deve continuar sendo WebRTC sobre UDP, enquanto a alternativa envia a mídia em frames binários pelo WebSocket? → A: WebRTC/UDP ou mídia binária via WebSocket, selecionados por configuração no frontend.
- Q: No modo WebSocket, como a tela deve ser codificada antes de ser enviada ao backend? → A: Fragmentos WebM/VP8 gerados com MediaRecorder.
- Q: Quando um espectador entrar durante uma transmissão WebSocket já iniciada, ele deve começar imediatamente usando um pequeno buffer temporário mantido pelo servidor? → A: Sim; manter somente os dados de inicialização e até 2 segundos recentes em memória.
- Q: A escolha entre `webrtc` e `websocket` deve ser fixa por ambiente/deploy ou selecionável pelo usuário na interface? → A: O deploy configura quais opções ficam disponíveis e o apresentador escolhe uma delas na tela para cada publicação.
- Q: O transporte WebSocket de mídia deve usar a mesma porta HTTP/HTTPS do backend ou uma porta separada? → A: A mesma porta HTTP/HTTPS, usando WebSocket seguro em produção.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Apresentador envia a tela ao servidor (Priority: P1)

Quem possui o token de apresentador inicia o compartilhamento como hoje,
mas sua tela é enviada para o serviço central. A quantidade de pessoas
assistindo não cria conexões diretas adicionais saindo do navegador do
apresentador.

**Why this priority**: retirar P2P exige primeiro estabelecer o servidor
como único destino da transmissão do apresentador.

**Independent Test**: criar um link, entrar com o token e iniciar a
captura; confirmar que o serviço recebe a transmissão e que o navegador
do apresentador não abre conexão de mídia com nenhum espectador.

**Acceptance Scenarios**:

1. **Given** um link válido e seu token, **When** o apresentador inicia
   o compartilhamento, **Then** a tela é enviada ao servidor e o estado
   da sala muda para transmitindo.
2. **Given** uma transmissão ativa, **When** novos espectadores entram,
   **Then** o apresentador continua enviando uma única transmissão ao
   servidor, sem criar uma conexão de mídia por espectador.
3. **Given** apenas o link público, **When** alguém tenta publicar mídia,
   **Then** o servidor recusa sem interromper a transmissão autorizada.
4. **Given** uma transmissão ativa, **When** o apresentador encerra ou
   fecha a sessão, **Then** o servidor para de aceitar/distribuir aquela
   mídia e a sala volta ao estado de espera.
5. **Given** qualquer um dos transportes suportados configurado,
   **When** o apresentador inicia a captura, **Then** uma única
   publicação chega ao servidor por WebRTC/UDP ou WebSocket binário.
6. **Given** mais de um transporte liberado pelo deploy, **When** o
   apresentador inicia o compartilhamento, **Then** escolhe na interface
   qual transporte aquela publicação utilizará.

---

### User Story 2 - Espectador recebe a tela do servidor (Priority: P1)

Quem abre o convite entra como espectador e recebe a tela exclusivamente
do serviço central. O navegador do espectador não se conecta diretamente
ao navegador do apresentador nem a outros espectadores.

**Why this priority**: a transmissão só entrega valor se o servidor
conseguir distribuí-la a quem recebeu o link.

**Independent Test**: iniciar uma transmissão, abrir o convite noutro
perfil e confirmar a tela; inspecionar as conexões e provar que o único
destino/origem de mídia do espectador é o servidor.

**Acceptance Scenarios**:

1. **Given** uma transmissão ativa, **When** um espectador abre o link,
   **Then** vê a tela enviada pelo servidor em menos de 5 segundos.
2. **Given** uma sala ainda em espera, **When** o espectador entra e o
   apresentador começa depois, **Then** a tela aparece sem recarregar.
3. **Given** uma transmissão ativa, **When** o apresentador encerra,
   **Then** o espectador volta à espera em menos de 5 segundos.
4. **Given** espectadores na mesma sala, **When** suas conexões são
   observadas, **Then** nenhum recebe endereço de rede, oferta de mídia
   ou fluxo pertencente diretamente a outro participante.
5. **Given** o mesmo transporte configurado no frontend e habilitado no
   backend, **When** o espectador entra, **Then** recebe a tela pelo
   servidor tanto no modo WebRTC/UDP quanto no modo WebSocket binário.

---

### User Story 3 - Vários espectadores sem multiplicar o envio do apresentador (Priority: P2)

O serviço redistribui a mesma tela para vários espectadores. Entradas,
saídas e falhas individuais não obrigam o apresentador a renegociar
conexões nem interrompem quem continua assistindo.

**Why this priority**: a principal vantagem do servidor intermediário é
desacoplar a transmissão do apresentador da quantidade de espectadores.

**Independent Test**: colocar 10 espectadores na mesma sala, entrar e
sair com alguns deles e confirmar que os demais continuam vendo a tela
e que o apresentador mantém uma única transmissão de saída.

**Acceptance Scenarios**:

1. **Given** uma transmissão ativa, **When** 10 espectadores entram,
   **Then** todos recebem a mesma tela pelo servidor.
2. **Given** vários espectadores conectados, **When** um deles sai ou
   perde conexão, **Then** os demais continuam assistindo sem
   interrupção perceptível.
3. **Given** espectadores entrando e saindo, **When** a presença muda,
   **Then** o contador reflete somente sessões atualmente conectadas.
4. **Given** nenhum espectador conectado, **When** o apresentador está
   transmitindo, **Then** a sessão permanece pronta para o próximo
   espectador sem criar conexão direta futura.

---

### User Story 4 - Recuperação e privacidade da transmissão (Priority: P2)

Se a conexão de mídia com o servidor oscilar, a interface informa a
situação e tenta recuperar a mesma sessão. A tela é apenas retransmitida
ao vivo: não é gravada nem mantida após o fim.

**Why this priority**: centralizar mídia aumenta a responsabilidade do
serviço sobre indisponibilidade e privacidade.

**Independent Test**: interromper temporariamente a conexão de mídia,
restaurá-la e confirmar recuperação; encerrar a transmissão e confirmar
que nenhum conteúdo reproduzível permanece disponível.

**Acceptance Scenarios**:

1. **Given** uma transmissão ativa, **When** o servidor perde a mídia do
   apresentador, **Then** espectadores veem estado de reconexão/espera,
   não um palco congelado indefinidamente.
2. **Given** uma interrupção temporária, **When** a conexão retorna,
   **Then** a mesma sala retoma a tela sem gerar novo convite.
3. **Given** o fim da transmissão, **When** alguém tenta obter quadros
   antigos, **Then** nenhum conteúdo gravado ou histórico está
   disponível.
4. **Given** uma falha interna na distribuição, **When** ela ocorre,
   **Then** o serviço registra diagnóstico da sala/sessão sem registrar
   token, conteúdo da tela ou segredo.

---

### Edge Cases

- O apresentador perde rede enquanto espectadores estão conectados:
  todos deixam de receber mídia e passam a reconexão/espera; ninguém é
  promovido a apresentador.
- O servidor reinicia durante uma transmissão: o link permanece válido,
  mas a mídia termina e precisa ser iniciada novamente pelo apresentador.
- Um espectador lento não pode bloquear a distribuição para os demais;
  sua qualidade pode degradar ou sua sessão pode ser encerrada.
- Dois publicadores tentam usar o mesmo link: somente a sessão autorizada
  como apresentador é aceita.
- Um espectador tenta enviar pacotes de mídia ou comandos de publicação:
  o servidor rejeita e mantém a pessoa como espectadora.
- A sala fica sem espectadores: o servidor pode descartar saídas de
  distribuição, mas mantém a entrada do apresentador enquanto a sessão
  estiver ativa.
- Um espectador entra durante uma publicação WebSocket: recebe os dados
  de inicialização e o buffer efêmero recente antes dos novos
  fragmentos ao vivo.
- Capacidade de mídia do servidor esgotada: novas publicações/assinaturas
  recebem erro claro; sessões já ativas não são silenciosamente
  substituídas.
- O transporte configurado no frontend não está habilitado no backend:
  a sessão recebe erro claro e não tenta conexão P2P nem troca
  silenciosamente de transporte.
- O proxy aceita HTTP, mas não upgrade de WebSocket: somente publicações
  WebSocket recebem erro de conexão claro; publicações WebRTC/UDP já
  ativas permanecem isoladas.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Toda mídia de tela MUST passar pelo servidor: apresentador
  → servidor → espectadores.
- **FR-002**: O sistema MUST NOT estabelecer conexão de mídia P2P entre
  apresentador e espectador nem entre espectadores.
- **FR-003**: O apresentador MUST enviar no máximo uma transmissão de
  tela ao servidor por link ativo, independentemente da quantidade de
  espectadores.
- **FR-004**: O servidor MUST redistribuir a transmissão autorizada para
  todos os espectadores conectados àquele link.
- **FR-005**: Somente a sessão validada com o token de apresentador MUST
  poder publicar mídia; espectadores MUST ser somente receptores.
- **FR-006**: Cada espectador MUST receber mídia somente da sala cujo link
  abriu; mídia e controle MUST NOT atravessar entre salas.
- **FR-007**: O link público, o identificador existente e as regras do
  token de apresentador MUST permanecer compatíveis com o produto atual.
- **FR-008**: Iniciar/parar mídia e entrar/sair MUST atualizar estado e
  presença sem recarregar ou gerar novo convite.
- **FR-009**: A saída de um espectador MUST remover sua distribuição no
  servidor e reduzir a presença sem interromper outros espectadores.
- **FR-010**: A queda do apresentador MUST encerrar a distribuição e
  retornar todos à espera; nenhum espectador MUST assumir publicação.
- **FR-011**: O servidor MUST NOT gravar, persistir ou disponibilizar
  reprodução da tela; a mídia existe somente durante a transmissão ao
  vivo.
- **FR-012**: Mídia e controle MUST ser protegidos em trânsito; tokens,
  conteúdo da tela e segredos MUST NOT aparecer em logs ou erros.
- **FR-013**: O sistema MUST informar estados distinguíveis de espera,
  conectando, transmitindo, reconectando e falha de mídia.
- **FR-014**: Uma sessão lenta ou com falha MUST ser isolada para não
  bloquear a distribuição aos outros espectadores.
- **FR-015**: O sistema MUST recusar novas sessões de mídia com erro claro
  quando o servidor não tiver capacidade, sem sequestrar ou substituir
  sessões ativas.
- **FR-016**: O transporte P2P e o handshake específico entre
  participantes (`ready`, ofertas/respostas direcionadas a session IDs)
  MUST ser removidos do fluxo público após a migração.
- **FR-017**: O backend MUST oferecer dois transportes de mídia
  servidor-mediado: WebRTC sobre UDP e frames binários por WebSocket.
- **FR-018**: O frontend MUST possuir uma configuração explícita para
  definir a lista de transportes disponíveis e o transporte padrão entre
  `webrtc` e `websocket`.
- **FR-019**: Os dois transportes MUST aplicar as mesmas regras de
  autorização, isolamento entre salas, mídia efêmera, capacidade e
  ausência de conexão P2P.
- **FR-020**: No modo `websocket`, o apresentador MUST gerar fragmentos
  WebM/VP8 com `MediaRecorder`; o backend MUST apenas validar e
  redistribuir esses fragmentos, sem transcodificação.
- **FR-021**: Durante uma publicação WebSocket, o backend MAY manter
  somente os dados de inicialização e no máximo 2 segundos recentes em
  memória para iniciar espectadores tardios; esse buffer MUST ser
  apagado ao parar, falhar ou reiniciar o serviço.
- **FR-022**: Quando o deploy disponibilizar mais de um transporte, a
  interface MUST permitir que o apresentador escolha o transporte antes
  de iniciar; o backend MUST associar a escolha à publicação e informar
  os espectadores, que MUST usar o mesmo transporte sem fallback
  automático.
- **FR-023**: A mídia WebSocket MUST usar a mesma porta HTTP/HTTPS do
  backend, com conexão segura em produção, sem exigir uma porta TCP
  adicional.

### Key Entities

- **Publicação de tela**: fluxo ao vivo único, autorizado pelo token e
  enviado pelo apresentador ao servidor para um link.
- **Distribuição de tela**: saída do servidor para um espectador
  conectado; pertence a uma publicação e termina quando a sessão sai.
- **Sessão de mídia**: vínculo temporário entre uma sessão da sala e o
  servidor, com papel publicador ou espectador, transporte selecionado e
  estado de conexão.
- **Capacidade de mídia**: limite operacional do servidor para aceitar
  publicações e distribuições simultâneas.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Em 100% dos testes de rede, nenhum participante estabelece
  conexão de mídia direta com outro participante; toda mídia tem o
  servidor como origem ou destino.
- **SC-002**: Em 95% das entradas sob rede estável, um espectador vê a
  tela em menos de 5 segundos após abrir uma transmissão ativa.
- **SC-003**: Uma sala suporta pelo menos 10 espectadores simultâneos,
  todos recebendo a mesma tela, enquanto o apresentador mantém uma única
  transmissão de saída.
- **SC-004**: Entrada ou saída de um espectador não interrompe os demais
  por mais de 1 segundo e não exige renegociação visível do apresentador.
- **SC-005**: Em 100% dos testes, espectador sem token não consegue
  publicar mídia nem receber mídia de outro link.
- **SC-006**: Ao parar a transmissão ou perder o apresentador, todos os
  espectadores voltam à espera em menos de 5 segundos.
- **SC-007**: Após encerrar uma transmissão, 0 segundos de conteúdo da
  tela ficam disponíveis para reprodução posterior.
- **SC-008**: Reiniciar o serviço encerra a mídia ativa, mas 100% dos
  links persistidos continuam abríveis para uma nova transmissão.
- **SC-009**: A suíte de aceitação executa publicação e visualização com
  `webrtc` e `websocket`; em ambos os modos, 100% da mídia passa pelo
  servidor e nenhuma conexão P2P é criada.
- **SC-010**: No modo WebSocket, um espectador tardio começa a reproduzir
  em menos de 5 segundos e nenhum fragmento permanece disponível após
  o encerramento da publicação.

## Assumptions

- Esta feature substitui integralmente o transporte P2P atual; não haverá
  modo híbrido nem fallback direto entre navegadores.
- O deploy define no frontend a lista permitida e o transporte padrão;
  quando ambos estão disponíveis, o apresentador escolhe na interface.
  Não há fallback automático entre os transportes.
- “UDP” no frontend significa WebRTC sobre UDP. Navegadores não abrem
  sockets UDP brutos.
- O modo WebSocket usa WebM/VP8 produzido pelo navegador; imagens
  sequenciais e protocolo WebCodecs customizado permanecem fora de
  escopo.
- O proxy reverso de produção encaminha o upgrade WebSocket na mesma
  origem HTTPS da API; somente a opção WebRTC exige a porta UDP de mídia.
- O link público, Base62 com salt, SQLite, token de apresentador, UI sem
  voz/câmera/chat e frontend embutido permanecem como nas features 001 e
  002.
- "Passar pelo servidor" significa que o servidor recebe a mídia do
  apresentador e produz uma saída para cada espectador. O plano escolherá
  o protocolo/componente de mídia compatível com essa regra.
- A tela continua somente vídeo; áudio, câmera, gravação, reprodução,
  controle remoto e chat permanecem fora de escopo.
- A mídia é efêmera e fica somente em memória/buffers necessários à
  retransmissão ao vivo; SQLite não armazena quadros ou gravações.
- O alvo inicial continua sendo pelo menos 10 espectadores simultâneos
  por link. Dimensionamento para centenas/milhares será uma feature
  posterior.
- Navegadores modernos em desktop e conexão segura em produção são o
  ambiente suportado.
- A constituição exige que novos canais/componentes de mídia sejam
  justificados no plano, mantendo Go/Gin em `api/`, Angular/Tailwind em
  `app/` e contratos públicos documentados.
