---
name: kb-search
description: "Search the local KB before project work and curate verified, reusable knowledge in the appropriate project or topic context."
---

# Habilidade: kb-search

Esta habilidade garante que a IA utilize a Base de Conhecimento (KB) local do desenvolvedor como fonte central de contexto e registro de aprendizados.

## Regras de Comportamento (MANDATORY)

1. **Fase de Pesquisa (Antes de tudo)**: Você DEVE pesquisar os termos principais relacionados ao pedido do usuário usando `cockpit kb search` ANTES de agir ou escrever códigos. Entenda o contexto e se há decisões arquiteturais ou problemas conhecidos.
2. **Curadoria**: Avalie se há conhecimento durável, verificado e útil para trabalho futuro. Quando houver, registre diretamente no contexto correto; não encerre toda resposta com uma sugestão de salvar. Quando não houver, não crie entrada nem mencione KB sem necessidade.

## Como usar a Busca

Execute o seguinte comando de CLI no terminal para realizar uma busca por termos específicos:

```bash
cockpit kb search "seu termo de busca"
```

### Exemplo de Uso (Busca):

Se o desenvolvedor solicitar para depurar as configurações de logs:
1. Execute: `cockpit kb search "logging"`
2. Analise os resultados retornados.
3. Se necessário, abra e leia o arquivo retornado utilizando a ferramenta de leitura de arquivos adequada.

## O que registrar

Registre decisões com justificativa, procedimentos reproduzíveis, causas confirmadas e soluções validadas que evitem retrabalho. Declare projeto/assunto, condições de aplicação, evidências e limitações. Separe fatos de hipóteses. Exemplo: uma falha de acesso ao keyring dentro de sandbox pode ser útil no KB do Cockpit; uma senha temporária e o resultado momentâneo de uma execução não são conhecimento para guardar.

Não registre conversas inteiras, todo pedido do usuário, status passageiro, duplicatas, credenciais ou suposições como fatos. Não misture assuntos de outros projetos no contexto `aicockpit`.

## Como salvar ou atualizar

1. Busque entradas relacionadas e leia a candidata antes de decidir criar outra.
2. Prepare Markdown com título descritivo, contexto, descoberta/decisão, procedimento e evidências relevantes. Evite copiar saídas com dados privados.
3. Para novo conhecimento do Cockpit: `cockpit kb add <arquivo.md> --context aicockpit`. Use `--root <diretório-do-kb>` quando houver um KB próprio do projeto; esse diretório será registrado para busca.
4. Para atualizar o mesmo assunto: `cockpit kb add <arquivo.md> --context <contexto> --root <raiz-original> --update <id>`. Preserve conteúdo anterior ainda válido no arquivo revisado: a atualização substitui o corpo inteiro.
5. Verifique o caminho/ID retornado e busque um termo específico da entrada. Erros não significam sucesso; se o arquivo foi salvo mas o índice falhou, corrija a indexação sem criar outra cópia.

Novas entradas usam `YYYY-MM-DD/YYYY-MM-DD_HHMMSS-assunto.md`, com horário local e offset nos campos `created` e `modified`. Atualizações mantêm caminho, ID e criação; apenas a modificação avança. Reimportar um arquivo que já está na raiz também preserva sua identidade. Reimportar uma fonte externa sem `--update` cria uma nova entrada: use o ID anterior para revisões.

A busca padrão é local, sem API ou Vault. Use `--graphify` somente quando quiser a extensão semântica configurada; uma falha dela não prova falha do KB local. Pastas datadas organizam criação, mas histórico completo de revisões exige versionar a raiz com Git ou backup.
