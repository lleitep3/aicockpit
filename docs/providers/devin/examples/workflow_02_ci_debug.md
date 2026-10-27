---
name: ci-debugger
description: Workflow to debug failing GitHub actions locally.
---
# CI Debugging
1. Read `.github/workflows/main.yml` to understand the failing step.
2. Replicate the environment variables locally using `.env.test`.
3. Run the exact failing bash step in an isolated Docker container: `docker run --rm -v $(pwd):/app -w /app node:18 bash -c "<failing_command>"`.
