---
name: pr-reviewer
description: Performs a comprehensive code review of the current Git diff before committing.
allowed-tools: [exec, read]
---
# PR Review Checklist
## Objective
Ensure code meets the security and style guidelines before submission.

## Execution
1. Run `git diff HEAD` to capture all staged and unstaged changes.
2. Check for the following anti-patterns:
   - Hardcoded API keys, secrets, or tokens.
   - Missing error handling in async operations (Promises/Try-Catch).
   - Any console.log statements left in the code.
3. If issues are found, do not commit. Instead, write a summary of the issues found to a `REVIEW_FEEDBACK.md` file.
