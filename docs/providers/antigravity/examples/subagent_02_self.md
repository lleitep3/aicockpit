# Subagent: self
type: self

Subagent that inherits the parent agent's full configuration including tools, system prompt, and model. 
Use this when you need to run an active task in a separate conversation context but with the same capabilities as the current agent.

Example Invocation (Tool Call):
```json
{
  "TypeName": "self",
  "Role": "Background Test Fixer",
  "Prompt": "Run `make test`. Identify the failing tests in internal/kb, fix the code, and verify tests pass. Do not touch cmd/."
}
```
