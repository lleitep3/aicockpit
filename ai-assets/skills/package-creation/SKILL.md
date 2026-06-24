---
name: package-creation
description: "Guides the AI agent on how to create, test, and publish modular packages for AICockpit."
---

# Habilidade: package-creation

Esta habilidade ensina as IAs a projetarem, desenvolverem e publicarem novos pacotes de extensões (plugins) para o AICockpit, seguindo as diretrizes e boas práticas oficiais.

## Quando Usar

Ative esta habilidade sempre que o desenvolvedor solicitar:
* A criação de um novo comando modular.
* A criação de um novo pacote de ferramentas ou regras.
* A publicação de uma extensão no cockpit-registry.

## Passo a Passo para Desenvolvimento

1. **Desenhar a Estrutura**:
   * Crie a pasta do pacote em `~/.cockpit/local-registry/<nome-do-pacote>`. **MANDATÓRIO**: Todo desenvolvimento e staging local de novos pacotes deve ser iniciado lá.
   * Crie o manifesto `cockpit-package.yml` especificando os metadados, features do módulo, regras, skills e os provedores de IA suportados.
   * Coloque os executáveis em `bin/` e as habilidades em `skills/`.

2. **Implementar o Comando wrapper (se necessário)**:
   * Desenvolva o script bash (ou binário compilado em Go) na pasta `bin/<nome-do-pacote>`.
   * Certifique-se de tratar argumentos de subcomandos (como `slice` na chamada de `video slice`) e repassar via switch case no script.
   * Dê permissão de execução ao script: `chmod +x bin/<nome-do-pacote>`.

3. **Boas Práticas de Prompt (Rules & Skills)**:
   * Sempre crie regras que estimulem o comportamento de otimização de tokens (ex: compressão de logs, fatiamento de vídeo).
   * As habilidades (`SKILL.md`) devem conter instruções precisas de chamadas da CLI e leitura de arquivos.

4. **Testes Locais (Não use `pkg install`!)**:
   * Devido à restrição de registros no cache remoto, **não** utilize `cockpit pkg install` para testar pacotes em desenvolvimento local.
   * Em vez disso, copie a pasta do seu pacote local de `~/.cockpit/local-registry/<nome-do-pacote>` diretamente para `~/.cockpit/packages/` e seus assets para as pastas de skills/rules correspondentes.
   * Atualize os workspaces com `cockpit deploy`.

5. **Publicação**:
   * Crie uma feature branch no repositório `cockpit-registry`.
   * Copie o diretório do pacote de `~/.cockpit/local-registry/<nome-do-pacote>` para a raiz da registry.
   * Registre o novo pacote no arquivo global `package-index.yaml` e envie as alterações.

## Referências Úteis
* Para detalhes arquiteturais e configuração avançada, leia o guia na Base de Conhecimento: [package-creation-publishing.md](file:///home/lleite/projects/aicockpit/ai-assets/knowledge-base/guides/package-creation-publishing.md) ou no padrão `~/.cockpit/kb/guides/package-creation-publishing.md`.
