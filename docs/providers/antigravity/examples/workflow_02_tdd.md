---
name: structured-tdd
description: Workflow for Test-Driven Development loops.
---
# Structured TDD Workflow
When the user requests implementing a feature via TDD:
1. Write the unit test in a `_test.go` file based on the specification.
2. Run the test. Verify it fails compilation or execution.
3. Write the minimal implementation required to pass the test.
4. Run the test again.
5. If it passes, refactor the implementation for clean code. Run the test again.
6. Stop and present the successful implementation to the user.
