# Antigravity Skills

Assim como o Goose e o Devin, o Google Antigravity também adotou o padrão unificado de **Agent Skills**. Por ser altamente autônomo, o Antigravity costuma utilizar essas skills delegando parte de suas capacidades para subagentes ou executando scripts encapsulados.

**Referência Oficial:** Uso interno (Google Antigravity SDK).

## Diretórios de Skills
- **Global:** `~/.gemini/config/skills/`
- **Global (por Plugin):** `~/.gemini/config/plugins/<nome-do-plugin>/skills/`

## Formato do Arquivo
Cada skill reside no seu próprio diretório contendo o arquivo `SKILL.md`. Em implementações mais avançadas, essas pastas contêm os subdiretórios adicionais `scripts/`, `examples/`, `resources/` e `references/` para embutir código fonte que a skill invoca.

## Padrão e Schema (YAML Frontmatter)
Exige-se um bloco YAML frontmatter no topo exato do `SKILL.md`. Sem os atributos `name` e `description` configurados no YAML, o modelo não mapeará a ferramenta internamente.

## Boas Práticas
1. **Strict Adherence (Aderência Estrita):** Os agentes Antigravity são explicitamente instruídos pela sua engine a ler todo o arquivo `SKILL.md` antes de prosseguir. Garanta que as instruções em markdown sejam imperativas e exatas.
2. **Tooling Extension:** Aproveite o diretório `scripts/` para anexar códigos nativos (em Python, Go, Bash). É muito mais confiável fazer o agente rodar o script associado a skill do que depender que ele escreva a solução inteira "de cabeça" (apenas via prompt).

## Exemplo de Skill

**Estrutura de Pastas:**
```text
skills/
└── web-scraper/
    └── SKILL.md
```

**Conteúdo do `SKILL.md`:**
```markdown
---
name: web-scraper
description: Extrai o conteúdo em texto de uma dada URL usando Python e BeautifulSoup. Use isso sempre que o usuário pedir para fazer scrape de um site ou resumir um artigo externo.
---

# Web Scraper Skill

Sempre que o usuário solicitar o scrape de uma página web, siga estes passos exatamente:

1. **Verify Environment:** Garanta que `beautifulsoup4` e `requests` estejam instalados. Caso contrário, instale-os usando `pip install beautifulsoup4 requests`.
2. **Execution:** Use um script Python para buscar a URL, extrair o texto da tag `<article>` (ou `<body>` se não existir article) e remover todas as marcações HTML.
3. **Output:** Retorne o texto puro para o usuário, prefixado com a frase: "Aqui está o conteúdo extraído:". Não retorne HTML puro.
```
