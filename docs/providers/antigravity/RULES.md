# Antigravity Rules

Para o Antigravity, as regras são chamadas de "Customizations" (ou Rulesets) e são lidas dinamicamente a cada sessão para guiar as decisões técnicas, restrições e padrões de comunicação do assistente.

**Referência Oficial:** Uso interno (Google Antigravity SDK).

## Diretórios e Arquivos de Regras
- **Global:** `~/.gemini/config/rules/AGENTS.md` (Pode ser também mapeado a partir de outros arquivos na pasta config/rules/ ou no raiz do config root). O Antigravity usa a raiz `~/.gemini/config`.
- **Project-level:** `.agents/AGENTS.md` na raiz do workspace.

## Formato do Arquivo
Arquivo em puro Markdown. Ao carregar o contexto, o sistema internaliza as diretivas escritas nos arquivos `AGENTS.md` que estiverem dentro das "Customization Roots".

## Boas Práticas
1. **Scope Management:** Adicione regras no `AGENTS.md` global APENAS se elas se aplicarem a absolutamente qualquer projeto e tarefa que você faça. Para guias de linguagem específicos, deixe na pasta de workspace `.agents/`.
2. **Priorização por Regras de Ouro:** Destaque regras inegociáveis no topo do arquivo. Se for crítico, o AICockpit costuma empacotar regras vitais num injetor automático (`gold_rules`) que garante que o Antigravity não as ignore.
3. **Padrões de Comunicação:** Use as regras do Antigravity também para formatar como ele se comunica com o usuário (ex: "Sempre liste os arquivos modificados em uma tabela no final").

## Exemplo de Regra

**Conteúdo do `AGENTS.md`:**
```markdown
# Antigravity General Guidelines

## 🤖 Segurança e Ferramentas
- Você está proibido de utilizar o comando `rm -rf` sem aprovação humana.
- Todo comando de shell deve utilizar o prefixo `rtk`.
- Se o CI falhar na verificação (por exemplo, durante a execução do comando de linting ou formatação do projeto), não tente fazer bypass; conserte o código e submeta novamente.

## 📝 Estilo e Respostas
- Responda de forma concisa e direta. 
- Mantenha o idioma em Português do Brasil.
- Ao usar o framework Cobra em Go, sempre trate os retornos de `RunE` adequadamente propagando os erros para o chamador.
```
