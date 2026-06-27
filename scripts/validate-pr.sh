#!/bin/bash

# Validate that a PR description follows the repository PR template.
# Usage: scripts/validate-pr.sh [file] (defaults to stdin)

set -euo pipefail

BODY_FILE="${1:-/dev/stdin}"
BODY=$(cat "$BODY_FILE")

ERRORS=0

require_section() {
  if ! grep -qF "$1" <<< "$BODY"; then
    echo "Error: PR description is missing section: $1" >&2
    ERRORS=$((ERRORS + 1))
  fi
}

require_section "## Descrição / Description"
require_section "## Tipo de Mudança / Type of Change"
require_section "## Impacto na Versão (Semantic Versioning)"
require_section "## Evidências / Evidence"
require_section "## Comandos para Teste / Test Commands"
require_section "## Checklist de Qualidade / Quality Checklist"

if ! grep -qE '\- \[x\].*(Bug fix|Nova Feature|Breaking change|Documentação|Refatoração|Configuração/CI)' <<< "$BODY"; then
  echo "Error: PR description must select at least one type of change" >&2
  ERRORS=$((ERRORS + 1))
fi

if ! grep -qE '\- \[x\].*(PATCH|MINOR|MAJOR)' <<< "$BODY"; then
  echo "Error: PR description must select at least one version impact" >&2
  ERRORS=$((ERRORS + 1))
fi

if ! grep -qE '\- \[x\]' <<< "$BODY"; then
  echo "Error: PR description must check at least one checklist item" >&2
  ERRORS=$((ERRORS + 1))
fi

if [ "$ERRORS" -gt 0 ]; then
  echo "Found $ERRORS validation error(s)." >&2
  exit 1
fi

echo "PR description follows the template."
