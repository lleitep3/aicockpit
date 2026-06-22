# Goose Skills

O Goose convergiu para o padrão aberto **Agent Skills**, tornando suas capacidades altamente compatíveis com outros agentes do mercado. Em vez de injetar todo o conhecimento num prompt gigante, o Goose carrega o conteúdo da skill dinamicamente com base em matching semântico.

**Referência Oficial:** [Goose Using Skills](https://goose-docs.ai/docs/guides/context-engineering/using-skills/)

## Diretórios de Skills
- **Global (Padrão Atual):** `~/.agents/skills/`
- **Global (Legado):** `~/.goose/skills/`
- **Project-level:** `.agents/skills/`

## Formato do Arquivo
Cada skill é representada por um diretório (ex: `minha-skill/`) contendo um arquivo principal chamado `SKILL.md`. Também pode incluir subdiretórios opcionais como `scripts/`, `references/` e `assets/`.

## Padrão e Schema (YAML Frontmatter)
A primeira seção do arquivo `SKILL.md` deve obrigatoriamente ser um cabeçalho YAML delimitado por `---`. Os campos exigidos são `name` e `description`.

## Boas Práticas
1. **Semantic Triggering (Gatilho Semântico):** Escreva o campo `description` de forma extremamente descritiva. O Goose usa essa descrição para determinar semanticamente quando deve acionar a skill correspondente.
2. **Progressive Disclosure (Revelação Progressiva):** Mantenha as instruções focadas. Em vez de colocar todo o conhecimento em um prompt imenso, modularize os workflows em skills, assim o Goose as carrega no contexto somente quando estritamente relevantes.

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
