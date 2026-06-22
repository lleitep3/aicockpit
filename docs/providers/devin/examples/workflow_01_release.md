---
name: release-workflow
description: Orchestrates the semantic release process.
---
# Release Workflow
1. Check that the working tree is clean.
2. Run `npm run lint` and `npm run test`.
3. If successful, run `npx standard-version` to bump the version and generate the CHANGELOG.md.
4. Push the new tag to origin: `git push --follow-tags origin main`.
