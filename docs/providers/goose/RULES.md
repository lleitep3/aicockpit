# Goose Rules

As regras no Goose (conhecidas nativamente como `hints`) atuam como o System Prompt contínuo do agente para uma determinada sessão ou workspace. Elas ditam comportamentos base, estilos de resposta e limitações estritas.

## Diretórios e Arquivos de Regras
- **Global:** `~/.config/goose/rtk-hints.md` (ou arquivos customizados injetados via configurações globais).
- **Project-level:** `.goosehints` na raiz do projeto.

## Formato do Arquivo
Arquivo Markdown puro. O Goose faz a concatenação textual dessas dicas/hints e as insere diretamente no contexto inicial das conversas.

## Boas Práticas
1. **Instruções Curtas e Imperativas:** Diferente de uma longa Wiki, o Goose processa melhor instruções diretas. Use "Sempre faça X", "Nunca faça Y".
2. **Uso de Tags Semânticas:** Em arquivos mais complexos, o Goose (e os LLMs por trás dele) lida muito bem com isolamento via Tags XML (ex: `<regras_de_codigo>`, `<limites_seguranca>`) dentro do markdown para modularizar o entendimento.
3. **Restrição de Ferramentas:** Use as regras do Goose se quiser proibir ativamente o uso de alguma ferramenta local ou fluxo não seguro antes mesmo dele tentar executar e falhar por permissão.

## Exemplo de Regra

**Conteúdo do `.goosehints`:**
```markdown
## 🏅 Gold Rules & Project Guidelines

> Estas regras são críticas e devem ser seguidas em todas as respostas e iterações.

<tool_restrictions>
- Sob nenhuma circunstância você deve rodar comandos destrutivos sem avisar.
- Todo e qualquer comando no terminal deve ser envelopado pelo binário `rtk`.
</tool_restrictions>

<workflow_rules>
### Antes de Commitar:
1. Revise se o arquivo modificado atende às convenções de Go do projeto.
2. Rode `make check`.
3. Se o passo 2 falhar, a ação de commit está bloqueada. Conserte os erros primeiro.
</workflow_rules>
```
