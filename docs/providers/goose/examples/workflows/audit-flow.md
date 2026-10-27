name: Security Audit
steps:
  - name: Lint
    run: golangci-lint run
  - name: Scan
    run: gosec ./...
  - name: Report
    run: echo 'Audit Complete'