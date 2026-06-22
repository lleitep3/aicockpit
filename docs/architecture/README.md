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

---
*Dica: Todos os documentos desta pasta utilizam diagramas interativos para facilitar o entendimento dos fluxos.*
