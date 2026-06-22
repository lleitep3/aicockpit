# Git Conventions

> Standards for commits, branches, and pull requests.

## Commit Messages — Conventional Commits

Format: `<type>(<scope>): <description>`

### Types
| Type | When to use |
|---|---|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `docs` | Documentation changes only |
| `style` | Formatting, missing semicolons (no logic change) |
| `refactor` | Code restructuring (no feature, no fix) |
| `test` | Adding or updating tests |
| `chore` | Build scripts, CI, dependencies, tooling |
| `perf` | Performance improvements |
| `ci` | CI/CD pipeline changes |
| `revert` | Reverts a previous commit |

### Examples
```
feat(auth): add JWT refresh token endpoint
fix(user): handle null email in profile update
docs(api): add OpenAPI examples for /users endpoint
test(auth): add integration tests for token expiry
chore(deps): upgrade fastapi to 0.111.0
refactor(service): extract UserValidator into separate class
```

### Rules
- **Subject line:** max 72 characters, imperative mood ("add" not "added"), no period at end
- **Body:** Use when more context is needed. Blank line between subject and body.
- **Scope:** Optional, refers to the module/component (e.g., `auth`, `user`, `api`)
- **Breaking changes:** Add `!` after type (`feat!:`) and explain in body with `BREAKING CHANGE:`

## Branch Strategy — GitHub Flow

```
main (always deployable)
  └── feature/<short-description>
  └── fix/<short-description>
  └── chore/<short-description>
  └── hotfix/<short-description>
```

### Rules
- **Never commit directly to `main`.** Always use a branch.
- **Branch names:** lowercase, kebab-case, prefixed with type
  - `feature/user-authentication`
  - `fix/null-email-crash`
  - `chore/upgrade-dependencies`
- **Short-lived branches:** Merge as soon as the feature is complete and reviewed
- **Delete after merge:** Clean up branches after merging

## Pull Requests

### PR Title
Follow the same format as conventional commits:
```
feat(auth): implement OAuth2 with Google
fix(api): correct 404 response format for missing users
```

### PR Description Template
```markdown
## What
Brief description of what this PR does.

## Why
Context and motivation. Link to issue if applicable. Closes #<issue-number>

## How
Key implementation decisions, if non-obvious.

## Testing
How to test this change manually (if applicable).

## Checklist
- [ ] Tests added/updated
- [ ] No new linting errors
- [ ] Documentation updated (if needed)
- [ ] No secrets committed
```

### Review Rules
- At least 1 approval before merging (solo projects: self-review)
- All CI checks must pass before merging
- Resolve all comments before merging
- Prefer squash merge for feature branches, merge commit for releases

## Tags and Releases

- Use semantic versioning: `v<MAJOR>.<MINOR>.<PATCH>`
- Tag releases on `main`
- `MAJOR`: breaking changes
- `MINOR`: new features, backward compatible
- `PATCH`: bug fixes, backward compatible
