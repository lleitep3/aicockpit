# 🏛️ AICockpit: Arquitetura do Sistema

Bem-vindo à documentação oficial da arquitetura do **AICockpit**. Esta pasta contém o aprofundamento técnico de como a ferramenta funciona por debaixo dos panos. 

Se você deseja desenvolver para o Cockpit, entender como pacotes são geridos, ou integrar uma nova IA provedora ao sistema, você está no lugar certo.

## 🛤️ Trilha de Leitura (Reading Trail)

Para garantir que você absorva o conteúdo da melhor forma sem se perder em jargões, preparamos uma trilha lógica de leitura. Recomendamos seguir esta ordem:

1. [**01. Visão Geral (High-Level Overview)**](01-overview.md)
   Entenda a anatomia básica do sistema. Como a linha de comando (`CLI`), a configuração (`config.yaml`), e as ferramentas (`hooks`, `skills`) se conectam em alto nível.
2. [**02. O Compilador Canônico e Provedores**](02-provider-compilers.md)
   Descubra como o AICockpit resolve o problema da fragmentação de Agentes de IA. Aprenda como a pasta unificada `.cockpit/` é dinamicamente "compilada" para os formatos nativos de IAs como Devin, Goose e Antigravity.
3. [**03. O Sistema de Pacotes**](03-package-system.md)
   Entenda o ciclo de vida de um pacote. Como o `PackageManager` faz o download, instala módulos e aciona os ganchos (*hooks*) de compilação.
4. [**04. Registros de Pacotes (Registries)**](04-package-registries.md)
   Aprofunde-se em como o ecossistema é distribuído. Como o Cockpit encontra pacotes, como funciona um arquivo `package-index.yaml`, e como você pode plugar seu próprio repositório de pacotes privado na sua empresa.
5. [**05. O Sistema de Cofre (Vault)**](05-vault-system.md)
   Gerenciamento seguro de credenciais, chaves de API e segredos que os agentes utilizam.
6. [**06. Knowledge Base Engine**](06-knowledge-base.md)
   Como o sistema indexa, busca e correlaciona o conteúdo semântico para os agentes locais.
7. [**07. Project & Task Management**](07-project-management.md)
   A máquina de estados para workflows, a UI do dashboard e a integração bi-direcional de tickets com o GitHub.

---
*Dica: Todos os documentos desta pasta utilizam diagramas interativos para facilitar o entendimento dos fluxos.*
