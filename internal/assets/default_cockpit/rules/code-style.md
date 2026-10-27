# Code Style

> Formatting and naming conventions for all projects.

## Python

### Formatting
- **Formatter:** Black (line length 88) or Ruff format
- **Linter:** Ruff (`ruff check .`) — replaces flake8, isort, pyupgrade
- **Type checker:** mypy or pyright in strict mode

### Naming
```python
# Modules and packages: snake_case
user_service.py
auth_utils.py

# Classes: PascalCase
class UserRepository:
class AuthService:

# Functions and variables: snake_case
def get_user_by_id(user_id: int) -> User:
current_user: User

# Constants: SCREAMING_SNAKE_CASE
MAX_RETRY_ATTEMPTS = 3
DEFAULT_TIMEOUT_SECONDS = 30

# Private: leading underscore
def _validate_token(token: str) -> bool:
```

### Structure (FastAPI projects)
```
src/
├── api/            # Route handlers (thin layer, no business logic)
│   └── v1/
├── services/       # Business logic
├── repositories/   # Data access layer
├── models/         # SQLAlchemy models
├── schemas/        # Pydantic schemas (request/response)
├── core/           # Config, deps, security
└── workers/        # Background tasks
```

### Imports
- Always use absolute imports (not relative `..`)
- Group: stdlib → third-party → local (ruff handles this automatically)
- No wildcard imports (`from module import *`)

## TypeScript / JavaScript

### Formatting
- **Formatter:** Prettier (default config)
- **Linter:** ESLint with TypeScript plugin

### Naming
```typescript
// Files: kebab-case
user-profile.tsx
auth-service.ts

// Components: PascalCase
function UserProfile({ userId }: UserProfileProps) {}

// Hooks: camelCase, prefixed with 'use'
function useUserProfile(id: string) {}

// Constants: SCREAMING_SNAKE_CASE
const MAX_ITEMS_PER_PAGE = 20;

// Types/Interfaces: PascalCase
interface UserProfileProps {
  userId: string;
}
type ApiResponse<T> = { data: T; error: null } | { data: null; error: string };
```

### Structure (React/Vite projects)
```
src/
├── components/     # Reusable UI components
├── pages/          # Route-level components
├── hooks/          # Custom hooks
├── services/       # API calls and external integrations
├── stores/         # State management (Zustand)
├── types/          # TypeScript type definitions
└── utils/          # Pure utility functions
```

### React
- Prefer functional components with hooks; no class components
- Colocate styles with components (CSS modules or styled components)
- Export types alongside components
- Use `React.FC` only when necessary; prefer explicit props typing

## General

### File organization
- One primary export per file
- Group related files in directories; use `index.ts` for re-exports
- Keep test files adjacent to source: `user.service.ts` → `user.service.test.ts`

### Comments
- **Why, not what.** Code explains what; comments explain why.
- Use `TODO:`, `FIXME:`, `HACK:`, `NOTE:` prefixes for special comments
- Remove commented-out code before committing

### Magic numbers
- No magic numbers in code. Extract to named constants with explanatory names.

```python
# Bad
time.sleep(300)

# Good
HEALTH_CHECK_INTERVAL_SECONDS = 300
time.sleep(HEALTH_CHECK_INTERVAL_SECONDS)
```
