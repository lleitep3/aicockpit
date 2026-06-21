# Subagent: research
type: research

Research subagent with read-only tools for exploring the codebase, searching the web, and reading files. 
Delegate to this agent when you need to run a research task in the background while continuing other work (e.g., coding, building, testing).

Example Invocation (Tool Call):
```json
{
  "TypeName": "research",
  "Role": "Codebase Researcher",
  "Prompt": "Find all usages of the 'Provider' struct in the internal/providers directory and summarize how the 'Skills' field is mapped."
}
```
