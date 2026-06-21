# AGENTS.md
This is the standard agent instruction file for the AcmeCorp Web Monorepo.

## Code Style
- We use strict TypeScript. `any` types are strictly forbidden.
- Always use functional components with React Hooks.
- CSS Modules are preferred over global stylesheets.

## Architecture
- `apps/web`: Next.js frontend frontend.
- `packages/ui`: Shared Radix UI components. Do not modify components here unless absolutely necessary.
- `packages/utils`: Shared pure functions.

## Commands
- To test: `pnpm run test`
- To lint: `pnpm run lint`
- To build: `pnpm run build`
