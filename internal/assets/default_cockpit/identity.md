# Developer Identity

> This file is the single source of truth about who I am as a developer.
> All AI tools will use this as their primary context when working with me.

## About Me

- **Name:** Leandro Leite
- **Role:** Full-stack developer / AI systems engineer
- **Location:** Brazil (BRT, UTC-3)
- **Primary language:** Portuguese (BR) for communication; English for code and documentation

## Development Central

**This project (`/home/lleite/projects/ai-cockpit`) is my central development hub.**

- All AI tools must understand this is the source of truth for my development setup
- When I say "let's create project X" or similar, always create the project directory in `/home/lleite/projects/`
- Example: "let's create project xpto" → create `/home/lleite/projects/xpto/`
- All new projects should reference this cockpit for rules, identity, and workflows
- This cockpit contains my developer identity, coding standards, git conventions, and AI agent configurations

## Tech Stack

### Backend

- **Primary:** Python (3.11+) with FastAPI
- **ORM:** SQLAlchemy (async preferred), Alembic for migrations
- **Task queues:** ARQ (asyncio-based), Celery when needed
- **Databases:** PostgreSQL (primary), Redis (cache/queue)
- **Testing:** pytest, pytest-asyncio, httpx for API tests

### Frontend

- **Primary:** React + TypeScript (strict mode)
- **Build:** Vite
- **Styling:** Vanilla CSS or Tailwind CSS (project-specific)
- **State:** Zustand or React Context (avoid Redux unless necessary)
- **Testing:** Vitest + React Testing Library

### AI / ML

- **Frameworks:** LangGraph, LangChain, AutoGen
- **Models:** Anthropic Claude, Google Gemini, OpenAI GPT
- **Agents:** Multi-agent orchestration patterns (orchestrator + specialists)
- **Protocols:** MCP (Model Context Protocol), WebSocket for real-time

### Infrastructure

- **Cloud:** GCP (primary), AWS (secondary)
- **Containers:** Docker + Docker Compose
- **CI/CD:** GitHub Actions
- **IaC:** Terraform (when needed)

### Tools

- **Version control:** Git + GitHub
- **Monorepo:** When applicable (Turborepo or simple scripts)
- **Package managers:** pip/uv (Python), npm/pnpm (Node)

## Architecture Preferences

- **API design:** RESTful with OpenAPI docs; WebSocket for real-time features
- **Async-first:** Prefer async/await in Python for I/O-bound operations
- **Separation of concerns:** Services, repositories, controllers clearly separated
- **12-factor app:** Environment variables, stateless processes, explicit config
- **Small, focused functions:** Functions should do one thing; prefer composition
- **Fail fast:** Validate inputs early, raise explicit errors, never swallow exceptions

## Work Style

- **TDD preferred:** Write tests before or alongside implementation, not after
- **Iterative:** Ship small, working increments; avoid big bang releases
- **Documentation:** Document why, not what; code should be self-explanatory
- **Code review:** Always review my own code before committing
- **Commits:** Atomic, conventional commits; each commit should pass CI

## Communication Preferences (for AI tools)

- **Be direct:** Skip preambles, get to the solution
- **Show, don't tell:** Prefer code examples over long explanations
- **Ask before deleting:** Always confirm before removing files or data
- **Explain trade-offs:** When multiple approaches exist, list pros/cons briefly
- **Portuguese or English:** Match the language I'm using in the session
