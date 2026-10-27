# Development Rules

> Universal rules that apply to ALL projects and ALL AI tools.
> These are non-negotiable behavioral guardrails.

## Safety Rules (Never Break These)

1. **Never delete files without explicit confirmation.** If a deletion is needed, ask first. Show what would be deleted and wait for approval.
2. **Never commit secrets.** No API keys, passwords, tokens, or credentials in any file. Use environment variables and `.env` files (git-ignored).
3. **Never push directly to `main` or `master`.** Always use a feature branch and PR.
4. **Never modify the database schema without a migration file.** Every schema change must have an Alembic (or equivalent) migration.
5. **Never break the test suite.** Run existing tests before declaring a task done. If a test fails, fix it or flag it — never ignore it.

## Code Quality Rules

6. **Write tests.** Every new function, endpoint, or component should have at least one test. Prefer writing tests before or alongside implementation.
7. **Handle errors explicitly.** Never use bare `except:` or swallow exceptions silently. Log errors with context; fail fast and loudly.
8. **Use type hints.** All Python functions must have type annotations. TypeScript must be in strict mode.
9. **No dead code.** Remove commented-out code before committing. Use version control instead of commenting out.
10. **Keep functions small.** If a function exceeds ~40 lines, consider splitting it. Single responsibility principle.

## Process Rules

11. **Read before writing.** Understand the existing code structure before adding new files. Check for similar patterns already in the codebase.
12. **Explain before executing.** For any non-trivial change, briefly state what you're about to do before doing it.
13. **One task at a time.** Complete and verify the current task before starting the next. Don't jump around between problems.
14. **Use the right tool.** Check if a library already exists before implementing from scratch. Prefer battle-tested solutions over custom implementations.
15. **Document decisions.** When making a significant architectural decision, add a brief comment or ADR explaining why.

## Environment Rules

16. **Use environment variables for config.** Never hardcode URLs, ports, credentials, or environment-specific values.
17. **Respect `.gitignore`.** Never commit `node_modules/`, `__pycache__/`, `.env`, build artifacts, or IDE-specific files.
18. **Virtual environments.** Always use a Python virtual environment (venv/uv). Never install packages globally.
19. **Docker for dependencies.** Use Docker Compose for local services (databases, queues, etc.). Don't require global installations of databases.

## AI Cockpit Rules

20. **When updating AI Cockpit, use `make install-local`.** After editing files in `canonical/`, always run `make install-local` to rebuild and distribute changes to all tools. Never run `python build.py` and `bash install.sh` separately.
