# Goose Permissions

O Goose adota o protocolo MCP (Model Context Protocol) para se conectar a servidores de ferramentas, CLIs e utilitários de sistema. Suas "permissões" geralmente ditam quais extensões estão habilitadas e quais variáveis de ambiente e restrições são injetadas em cada provedor de skill.

**Referência Oficial:** [Goose Extensions & Config](https://goose-docs.ai/)

## Diretórios e Arquivos de Permissões
- **Global:** `~/.config/goose/config.yaml`
- **Project-level:** (Geralmente gerido pelas flags de ambiente via CLI ou extensões locais na pasta do projeto).

## Formato do Arquivo
Arquivo YAML gerenciando os perfis (profiles) e a lista de extensões (extensions).

## Padrão e Schema
```yaml
extensions:
  developer:
    enabled: true
    cmd: "npx"
    args: ["-y", "@goose-ai/developer-extension"]
  cockpit-rtk:
    enabled: true
    cmd: "rtk"
    args: ["mcp", "start"]
```

## Boas Práticas
1. **Controle de Extensões:** Não instale extensões MCP ou ferramentas que você não confia integralmente, uma vez que o Goose poderá invocá-las de forma autônoma.
2. **Separação de Identidades:** Use o arquivo de configuração para isolar perfis. Você pode habilitar ou desabilitar conjuntos de ferramentas baseado na tarefa a ser resolvida pelo Goose, alterando as permissões indiretamente ativando/desativando integrações no `config.yaml`.
