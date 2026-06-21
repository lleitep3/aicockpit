# AGENTS.md (Project Rules)
## AICockpit Core Guidelines
- Language: Go 1.26+ (MANDATORY).
- CLI Framework: Cobra (`github.com/spf13/cobra`).
- Minimum Test Coverage: 90%. This is MANDATORY for all PRs.
- Error Handling: Do not ignore errors using `_`. Always return wrapped errors using `fmt.Errorf("...: %w", err)`.
