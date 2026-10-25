# GitHub Copilot CLI provider

O provider `github-copilot` do AICockpit integra os arquivos canônicos com o CLI oficial do GitHub Copilot. A integração é global: o Cockpit publica arquivos em `~/.copilot`, e o CLI os carrega em diferentes repositórios.

## Artefatos publicados

| Conteúdo canônico | Destino do Copilot CLI | Observação |
| --- | --- | --- |
| Entrypoint e regras | `~/.copilot/copilot-instructions.md` | O adapter substitui apenas o bloco `cockpit:global` e preserva texto fora dele. |
| Skills | `~/.copilot/skills/<name>/SKILL.md` | Scripts auxiliares do skill acompanham o arquivo. |
| Workflows | `~/.copilot/skills/cockpit-workflow-<name>/SKILL.md` | Checklists viram skills invocáveis. Isso não cria um dynamic workflow do Copilot, que exige uma extensão com código. |
| Agentes | `~/.copilot/agents/<name>.agent.md` | `AllowedTools` vira a lista `tools`; o agente inclui instruções do repositório quando roda como subagente. |

O provider está habilitado por padrão e detecta a instalação pelo executável `copilot` no `PATH`.

## Limites

- O provider não gera política de permissões. Aprovações do Copilot CLI são associadas ao repositório e não representam por completo as regras canônicas `allow`, `deny` e `ask` do Cockpit. A feature `permissions` fica desabilitada; se ativada com uma política não vazia, o deploy retorna erro em vez de gravar um arquivo que o CLI ignora.
- `CanonicalAgent.Permissions` e `MaxNesting` também não podem ser traduzidos para um agente do Copilot CLI. O deploy falha para esse agente. Use `AllowedTools` para definir uma lista positiva de ferramentas.
- Workflows canônicos são checklists acionáveis por skill. Dynamic workflows do Copilot são extensões executáveis e não são produzidos pelo adapter.
- Hooks, MCP e configurações do próprio CLI não são escritos pelo provider.

## Deploy e validação

Depois de configurar regras, skills, workflows e agentes no Cockpit, rode `cockpit deploy`. O comando publica os artefatos de todos os providers ativos na configuração.

Para adicionar ao Cockpit uma skill local que o Copilot já encontra, passe a pasta da skill ou a pasta que contém várias skills:

```bash
cockpit skills import ~/.agents/skills --dry-run
cockpit skills import ~/.agents/skills --save-conflicts ~/.cockpit/skill-reviews/copilot-local
```

Também funciona com `~/.copilot/skills` ou uma pasta de projeto, como `.github/skills`. O importador preserva `SKILL.md`, metadados YAML, arquivos auxiliares em subpastas e permissões de scripts. Skills com o mesmo nome não são sobrescritas: idênticas são marcadas como `Unchanged`; diferentes aparecem como `Conflict` e ficam intactas. `--save-conflicts <nova-pasta>` guarda cópias dos conflitos fora da biblioteca canônica, para revisão e merge posterior; essa pasta não é publicada por `cockpit deploy` e precisa ainda não existir. Após importar skills novas, rode `cockpit deploy` para publicar a biblioteca nos providers ativos. `--json` gera um relatório para automação.

Confirme a descoberta pelo CLI:

```bash
copilot instruction list
copilot skill list
```

O provider não autentica o usuário nem altera credenciais do GitHub Copilot.

## Referências oficiais

- [Instruções personalizadas do Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions)
- [Skills do Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-skills)
- [Agentes personalizados do Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli)
- [Diretório e permissões do Copilot CLI](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference)
