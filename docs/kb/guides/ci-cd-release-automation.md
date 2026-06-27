# Automação de CI/CD: Changelog, Release e Proteção da Main

## Contexto

O repositório `lleitep3/aicockpit` usa a branch `main` protegida. A proteção exige:

- Pull request obrigatório
- Assinatura de commits (`required signed commits`)
- Status checks passando (`lint`, `test`, `vulnerabilities`, `CodeQL`)

Por causa disso, o workflow antigo de changelog/release — que fazia push direto na `main` — parou de funcionar. Foi necessário migrar para um fluxo baseado em PRs criados por um bot de CI, com merge administrativo bypassando a assinatura.

---

## Arquitetura Atual

### Workflows principais

#### `update-changelog.yml`

- Gatilho: `push` na `main` e `workflow_dispatch`
- O job `update-changelog`:
  1. Faz checkout completo com `RELEASE_TOKEN`
  2. Roda `scripts/update-changelog.sh --pr`
  3. O script detecta se há mudanças desde a última tag
  4. Se houver, gera um novo `CHANGELOG.md`, cria um branch `chore/changelog-update-...` e abre um PR
  5. Faz merge do PR com `gh pr merge --admin --delete-branch`

#### `release.yml`

- Gatilho: `workflow_run` quando o workflow `Update Changelog` completa com sucesso
- O job `release`:
  1. Faz checkout com `RELEASE_TOKEN`
  2. Roda `scripts/bump-release.sh --pr`
  3. Determina o bump (MAJOR/MINOR/PATCH) a partir dos commits Conventional Commits
  4. Atualiza `VERSION`, `internal/version/version.go` e `CHANGELOG.md`
  5. Cria PR `chore/release/bump-vX.Y.Z` e faz merge com `--admin`
  6. Cria tag `vX.Y.Z` e GitHub Release

#### `pr-validation.yml`

- Gatilho: `pull_request` em `main` e `develop`
- Valida se a descrição do PR segue o template obrigatório (seções, tipo de mudança, impacto de versão, checklist)
- Roda `scripts/validate-pr.sh`

#### `pr-changelog.yml`

- Gatilho: `pull_request` em `main` e `develop`
- Gera um preview do changelog para o PR

### Scripts

#### `scripts/generate-changelog.sh`

- Lê commits entre a última tag e `HEAD`
- Separa por tipo (`feat`, `fix`, `docs`, `refactor`, `chore`, `ci`, etc.)
- Gera conteúdo no formato keepachangelog
- Regex para capturar `type(scope): description` e `type: description`

#### `scripts/update-changelog.sh`

- Modos: `--dry-run` e `--pr`
- Em `--pr`:
  - Cria branch com timestamp
  - Gera e commita `CHANGELOG.md` com `[skip ci]` no título
  - Abre PR com `RELEASE_TOKEN`
  - Faz merge com `--admin` para bypassar a exigência de assinatura

#### `scripts/bump-release.sh`

- Determina bump:
  - `feat(...)!:` ou `BREAKING CHANGE` → MAJOR
  - `feat(...)` → MINOR
  - `fix(...)` → PATCH
  - Outros → PATCH
- Atualiza `VERSION`, `internal/version/version.go`, `CHANGELOG.md`
- Cria PR com `[skip ci]` e faz merge com `--admin`
- Cria tag e release

---

## Configuração Necessária

### 1. Branch protection da `main`

- `Require a pull request before merging`
- `Require signed commits`
- `Require status checks to pass`: `lint`, `test`, `vulnerabilities`, `CodeQL`
- `Require code owner reviews` (opcional, mas recomendado para `.github/`)

### 2. Secret `RELEASE_TOKEN`

- Criar um Personal Access Token (classic) com escopo `repo` (privado) ou `public_repo` (público)
- Ou fine-grained token com `Contents: Read and write` e `Pull requests: Read and write`
- Adicionar como repository secret `RELEASE_TOKEN`

### 3. Permissão do GitHub Actions

- Settings → Actions → General → Workflow permissions
  - `Read and write permissions`
  - `Allow GitHub Actions to create and approve pull requests`

### 4. CODEOWNERS

- `.github/CODEOWNERS`:
  ```
  .github/ @lleitep3
  ```
- Isso exige aprovação do dono para mudanças em workflows

---

## Fluxo Completo (exemplo)

1. Usuário mergeia um PR com commit `feat(metrics): add new collector`
2. Push na `main` dispara `Update Changelog`
3. Workflow cria PR `docs(changelog): update CHANGELOG.md for changes since vX.Y.Z [skip ci]`
4. PR é mergeado com `--admin`
5. Merge do changelog dispara `Release` via `workflow_run`
6. Workflow cria PR `chore(release): bump version and update CHANGELOG.md to vX.Y.Z [skip ci]`
7. PR é mergeado com `--admin`
8. Tag `vX.Y.Z` e GitHub Release são criados

---

## Problemas Encontrados e Soluções

### PRs criados por `GITHUB_TOKEN` não disparam checks

- Causa: eventos gerados por `GITHUB_TOKEN` não disparam novos workflows
- Solução: usar `RELEASE_TOKEN` (PAT) para criar e mergear os PRs

### Commits do bot não são assinados

- Causa: a proteção de branch exige assinatura de commits
- Solução: merge com `gh pr merge --admin` usando o `RELEASE_TOKEN` do dono do repo

### `[skip ci]` impede que checks rodem no PR do bot

- Causa: o commit do changelog tem `[skip ci]`, então os workflows de pull request não rodam
- Solução: não esperar checks; fazer merge direto após criação do PR

### `setup-go@v4` deprecated

- Causa: versão antiga do action
- Solução: atualizar para `actions/setup-go@v5`

### Heredoc em YAML causava erro de parsing

- Causa: heredoc dentro de `run` no workflow
- Solução: mover lógica para scripts `.sh` e chamar do YAML

---

## Boas Práticas

- Sempre usar Conventional Commits (`feat(scope):`, `fix(scope):`, etc.)
- O tipo de commit determina o bump de versão
- Não fazer push direto na `main`
- Manter `RELEASE_TOKEN` válido e com escopo adequado
- Não remover `[skip ci]` dos commits automatizados de changelog/release
- Atualizar a documentação de CI/CD quando o workflow mudar

---

## Comandos Úteis

```bash
# Dry-run do changelog
bash scripts/update-changelog.sh --dry-run

# Dry-run do release
bash scripts/bump-release.sh --dry-run

# Validar descrição de PR
bash scripts/validate-pr.sh <<'EOF'
## Descrição / Description
...
EOF

# Forçar rerun do changelog
gh workflow run update-changelog.yml --ref main

# Listar releases
gh release list
```

---

## Referências

- `.github/workflows/update-changelog.yml`
- `.github/workflows/release.yml`
- `.github/workflows/pr-validation.yml`
- `scripts/update-changelog.sh`
- `scripts/bump-release.sh`
- `scripts/generate-changelog.sh`
- `scripts/validate-pr.sh`
