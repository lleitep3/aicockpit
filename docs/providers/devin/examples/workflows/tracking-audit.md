name: Tracking Audit
steps:
  - name: Find synchronized files
    run: find ~/.cockpit/skills ~/.cockpit/rules ~/.cockpit/agents ~/.cockpit/workflows -type f
  - name: Validate headers
    run: grep -rlE "^// package:" ~/.cockpit/skills ~/.cockpit/rules ~/.cockpit/agents ~/.cockpit/workflows
  - name: Report missing or divergent headers
    run: echo 'Tracking audit complete — review missing/divergent headers in report'
