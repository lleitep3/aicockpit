---
name: workflow-skill-creator
description: Distills a completed user workflow or interaction into a reusable agent skill.
---
# Workflow Skill Creator
Use when the user asks to turn their workflow, interaction, or multi-step process into a skill.

1. Analyze the transcript of the recent successful workflow.
2. Extract the exact shell commands, code patterns, and logic used.
3. Generate a new `SKILL.md` file in `~/.gemini/config/skills/<new_skill_name>/`.
4. The new skill becomes a permanent workflow that the agent can trigger in the future.
