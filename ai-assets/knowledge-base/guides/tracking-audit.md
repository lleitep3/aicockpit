# Guia de Auditoria de Tracking

Este guia descreve como auditar o rastreamento de proveniência em arquivos sincronizados pelo AICockpit.

## Objetivo da Auditoria

Garantir que todo arquivo gerado ou sincronizado por um pacote Cockpit contenha um cabeçalho (`tracking header`) com metadados corretos e que esses metadados reflitam o pacote real de origem.

## Cabeçalho Esperado

Um tracking header válido segue o formato:

```text
// package:<nome> version:<versão> created:<data> updated:<data>
```

Exemplo:

```text
// package:hello-world version:1.0.0 created:2026-01-01T00:00:00Z updated:2026-01-02T00:00:00Z
```

## Campos Obrigatórios

| Campo       | Validação                                              |
|-------------|--------------------------------------------------------|
| `package`   | Deve coincidir com `pkg.Name` do pacote de origem      |
| `version`   | Deve coincidir com `pkg.Version`                       |
| `created`   | Deve coincidir com `pkg.Metadata.CreationDate`         |
| `updated`   | Deve coincidir com `pkg.Metadata.LastModified`         |

## Passos da Auditoria

1. **Listar arquivos sincronizados**
   - Identifique a pasta canônica (ex: `~/.cockpit/skills`, `~/.cockpit/rules`, `~/.cockpit/agents`, `~/.cockpit/workflows`).
   - Filtre arquivos que deveriam ter tracking header.

2. **Verificar presença do header**
   - Leia as primeiras linhas do arquivo.
   - Confirme se a primeira linha inicia com `// package:`.

3. **Extrair e validar campos**
   - Extraia os valores com uma regex simples: `// package:(\S+) version:(\S+) created:(\S+) updated:(\S+)`.
   - Compare com `cockpit-package.yml` do pacote instalado.

4. **Verificar integridade do pacote de origem**
   - Localize o pacote em `~/.cockpit/packages/<package>`.
   - Leia `cockpit-package.yml` e confirme `name`, `version`, `creation_date` e `last_modified`.

5. **Reportar anomalias**
   - Header ausente.
   - Campos divergentes entre header e manifesto.
   - Arquivos sem pacote de origem correspondente.

## Ferramentas Disponíveis

- `internal/tracking.GenerateHeader(pkg)` — gera o header de um pacote.
- `internal/tracking.InjectHeader(filePath, pkg)` — insere ou atualiza o header de um arquivo.
- `scripts/tracking-audit.sh` — script de audit (quando implementado).

## Exemplo de Uso

```bash
# Gera o header de um pacote
cockpit tracking generate --package hello-world

# Injeta o header em um arquivo
cockpit tracking inject skills/hello/SKILL.md --package hello-world
```

## Ações Corretivas Comuns

| Problema                  | Ação                                              |
|---------------------------|---------------------------------------------------|
| Header ausente            | Reexecutar `InjectHeader` com o pacote correto    |
| Versão desatualizada      | Atualizar para `pkg.Version` atual                |
| Campos mal formatados     | Verificar `packages.SavePackage` e `metadata`     |
| Arquivo órfão             | Remover ou associar a pacote correto              |
