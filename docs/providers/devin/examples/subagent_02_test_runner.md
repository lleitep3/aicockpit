# Delegating E2E Tests
E2E tests take 15 minutes to run. Do not run them in the main thread.

Example Trigger:
```bash
delegate "Run `pnpm run test:e2e`. If it fails, extract the exact error message from Playwright and return it to me."
```
