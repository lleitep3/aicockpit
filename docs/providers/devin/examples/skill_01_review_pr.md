---
name: review-pr
description: Review staged changes for code quality and adherence to style guide
allowed-tools:
  - read
  - grep
  - glob
  - exec
permissions:
  allow:
    - Exec(git diff)
    - Exec(git log)
---

# PR Review

Review the current staged changes for quality issues:

!`git diff --staged`

Evaluate the changes based on:

1. **Correctness** — Any logic errors or edge cases?
2. **Security** — Any vulnerabilities introduced?
3. **Performance** — Any obvious inefficiencies?
4. **Style** — Consistent with the codebase conventions?
5. **Testing** — Are tests included and adequate?

Provide a summary with:
- Specific line references for any issues found
- Suggestions for improvements
- Overall assessment (approve/needs changes)

If the changes look good, indicate that the PR is ready for merge.
