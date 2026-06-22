name: Release Pipeline
steps:
  - name: Test
    run: make test
  - name: Build
    run: make build
  - name: Deploy
    run: ./deploy.sh