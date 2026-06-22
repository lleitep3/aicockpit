# 01. Visão Geral (High-Level Overview)

O **AICockpit** é um framework de linha de comando (`CLI`) escrito em **Go (Golang)**. Seu objetivo principal é fornecer uma interface unificada para ferramentas, *skills*, e bases de conhecimento para Agentes de IA autônomos.

Em vez de você ter que gerenciar habilidades separadas para diferentes modelos de IA (ex: Devin, Goose, Antigravity, Claude), o AICockpit serve como a "ponte" que orquestra tudo a partir de um único padrão central.

## Arquitetura Macro

A arquitetura do Cockpit é dividida em 3 grandes domínios:

1. **A Interface (CLI):** O comando `cockpit` ou `rtk`. Interage com o desenvolvedor.
2. **O Core Interno (Engine):** Gerencia as regras de negócio, configurações, compilações e log de métricas.
3. **O Ecossistema Externo:** Os agentes de IA, os repositórios de pacotes (Registries) e a máquina local onde a execução ocorre.

Abaixo temos um diagrama visual simplificado do fluxo do sistema:

```mermaid
graph TD
    User([Usuário / Desenvolvedor]) --> CLI[AICockpit CLI]
    AI([Agentes de IA]) --> CLI
    
    subgraph AICockpit Core
        CLI --> Config[Config Manager\n(config.yaml)]
        CLI --> PkgMgr[Package Manager]
        CLI --> KB[Knowledge Base\nSearch Engine]
        
        PkgMgr --> ProvMgr[Provider Manager\n(Canonical Compiler)]
    end
    
    subgraph Ecossistema Externo
        PkgMgr -- "Faz o download de" --> Registries[(Package Registries)]
        KB -- "Lê e Indexa" --> LocalFiles[Arquivos Markdown Locais]
        ProvMgr -- "Injeta Skills em" --> AIFolders[(Pastas de IA\n.gemini, .devin)]
    end
```

### Componentes Principais

* **Config Manager (`internal/config`):** Lê e grava as configurações do sistema baseadas no arquivo `~/.cockpit/config.yaml`. Armazena preferências de idioma, telemetria e configurações do *Knowledge Base*.
* **Package Manager (`internal/packages`):** O "NPM" ou "APT" do Cockpit. Gerencia a busca, download, instalação e desinstalação de pacotes.
* **Provider Manager (`internal/providers`):** O motor de compilação. Traduz os pacotes baixados em configurações ativas nos agentes de IA. *(Discutiremos profundamente na próxima etapa).*
* **Knowledge Base Engine (`internal/kb`):** O motor de busca vetorial/keyword que as IAs utilizam para encontrar contexto dentro dos arquivos do próprio repositório.
* **Logger & Metrics (`internal/logging`):** Tudo que passa pelo CLI é logado. Isso permite analisar a performance das IAs e economizar *tokens*.

### O Ciclo de Vida de uma Execução

Quando um humano ou uma IA executa `cockpit kb search "metrics"`:
1. O executável Go processa os argumentos via framework `Cobra`.
2. A configuração e idioma (Inglês ou Português) são carregados do `~/.cockpit/config.yaml`.
3. O comando de Busca do KB é acionado passando o termo.
4. O motor lê os arquivos do cache (`.index.json`) para ganhar velocidade.
5. Os resultados são parseados e formatados no terminal.
6. A execução é logada no motor de Métricas (`metrics.json`).

> **Próximo Passo:** Agora que você tem a visão geral de como o sistema está acoplado, vá para [02. O Compilador Canônico e Provedores](02-provider-compilers.md) para entender a mágica da compatibilidade multi-IA.
