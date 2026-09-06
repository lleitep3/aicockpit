# Variáveis de ambiente do AICockpit

Este arquivo é a referência única das variáveis consumidas pelo núcleo do Cockpit. Variáveis específicas de integrações, como `GITHUB_TOKEN`, devem ser declaradas e documentadas pelo pacote que as utiliza.

| Variável | Componente | Finalidade | Sensível |
|---|---|---|---|
| `COCKPIT_DATA_DIR` | Core | Raiz compartilhada para dados persistentes e artefatos de execução | Não |
| `COCKPIT_LOG_DIR` | Core/logging | Substitui explicitamente o diretório de logs | Não |
| `TRACKING_DIR` | Compatibilidade/tracking | Substitui o diretório legado de tracking | Não |
| `COCKPIT_LANGUAGE` | Core | Sobrescreve o idioma | Não |
| `COCKPIT_VERSION` | Core | Sobrescreve a versão reportada | Não |
| `COCKPIT_DEV_MODE` | Desenvolvimento | Habilita comportamentos de desenvolvimento | Não |
| `COCKPIT_APP_ID` | Vault | Define o namespace da aplicação | Não |
| `COCKPIT_TEST_GIT` | Testes | Substitui o executável Git em testes | Não |

## Regras

- O core não deve ler credenciais de integrações. Por exemplo, `GITHUB_TOKEN` pertence ao pacote `github-tools`.
- Caminhos configuráveis devem ser validados com uma tentativa real de escrita e ter fallback para o diretório temporário do sistema.
- Variáveis de ambiente não devem conter segredos quando o Vault puder ser usado.
- Pacotes devem declarar suas variáveis em `requirements.environment` no manifesto `cockpit-package.yml`.

Para detalhes do fallback de logs, consulte o [troubleshooting de ambiente do shell](../kb/troubleshooting/shell-environment.md).
