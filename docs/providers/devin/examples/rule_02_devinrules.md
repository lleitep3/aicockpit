---
trigger: "**/tests/*.spec.ts"
---
# Test File Rules
When editing Playwright test files in this repository:
1. Always use `test.step` for logical grouping of actions.
2. Assertions must use `expect(page.locator(...)).toBeVisible()` rather than checking DOM state manually.
3. Do not use hardcoded wait times (`page.waitForTimeout`); always wait for network states or element visibility.
