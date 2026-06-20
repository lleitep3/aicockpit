# AICockpit SDLC (Software Development Lifecycle)

## Visão Geral
Este documento define o processo de desenvolvimento para o projeto AICockpit, garantindo qualidade, consistência e boas práticas de Go.

## 1. Padrões de Código Go

### 1.1 Formatação e Linting
- **go fmt**: Formatação automática (executado antes de cada commit)
- **goimports**: Organização automática de imports
- **golangci-lint**: Análise estática abrangente

### 1.2 Convenções de Nomenclatura
- Pacotes: lowercase, sem underscores (`config`, `logger`, não `config_manager`)
- Funções públicas: PascalCase (`NewLogger`, `LoadConfig`)
- Funções privadas: camelCase (`setupLogger`, `loadConfig`)
- Constantes: UPPER_SNAKE_CASE (`DEFAULT_LOG_LEVEL`)
- Variáveis: camelCase (`logLevel`, `configPath`)

### 1.3 Estrutura de Pacotes
```
aicockpit/
├── cmd/                    # Comandos CLI
│   ├── root.go
│   ├── setup.go
│   ├── doctor.go
│   └── info.go
├── internal/               # Pacotes internos (não exportáveis)
│   ├── config/            # Gerenciamento de configuração
│   ├── logger/            # Sistema de logging
│   ├── i18n/              # Internacionalização
│   └── executor/          # Execução de comandos
├── pkg/                    # Pacotes públicos (se necessário)
├── tests/                  # Testes de integração
├── main.go                # Ponto de entrada
├── go.mod
├── go.sum
├── Makefile               # Automação de tarefas
├── .golangci.yml          # Configuração do linter
└── SDLC.md               # Este arquivo
```

## 2. Workflow de Desenvolvimento

### 2.1 Antes de Implementar
1. Criar uma issue/task descrevendo o que será feito
2. Criar uma branch: `git checkout -b feature/nome-descritivo`
3. Atualizar o TODO list do projeto

### 2.2 Durante a Implementação
1. **Escrever código** seguindo as convenções
2. **Testar localmente**: `make test`
3. **Validar código**: `make lint`
4. **Formatar código**: `make fmt`
5. **Build**: `make build`

### 2.3 Depois de Implementar
1. **Executar suite completa**: `make check` (lint + test + build)
2. **Revisar mudanças**: `git diff`
3. **Commit com mensagem clara**
4. **Atualizar TODO list**

## 3. Comandos Make (Automação)

```makefile
.PHONY: help build test lint fmt check clean install

help:
	@echo "AICockpit - Available commands:"
	@echo "  make build      - Build the binary"
	@echo "  make test       - Run tests"
	@echo "  make lint       - Run linters"
	@echo "  make fmt        - Format code"
	@echo "  make check      - Run all checks (lint + test + build)"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make install    - Install binary locally"

build:
	go build -o bin/cockpit .

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...
	goimports -w .

check: fmt lint test build
	@echo "✓ All checks passed!"

clean:
	rm -rf bin/ coverage.out

install: build
	cp bin/cockpit $(GOPATH)/bin/
```

## 4. Testes

### 4.1 Estratégia de Testes
- **Unit Tests**: Para funções e métodos individuais
- **Integration Tests**: Para fluxos completos
- **Coverage Target**: Mínimo 70% para novos código

### 4.2 Estrutura de Testes
```
internal/
├── config/
│   ├── config.go
│   └── config_test.go
├── logger/
│   ├── logger.go
│   └── logger_test.go
```

### 4.3 Exemplo de Teste
```go
package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	
	if cfg.Version == "" {
		t.Error("Version should not be empty")
	}
}
```

## 5. Commits e Versionamento

### 5.1 Mensagens de Commit
```
<tipo>: <descrição breve>

<descrição detalhada (opcional)>

Closes #<issue-number>
```

Tipos:
- `feat`: Nova feature
- `fix`: Correção de bug
- `refactor`: Refatoração
- `test`: Adição/modificação de testes
- `docs`: Documentação
- `chore`: Tarefas administrativas

### 5.2 Versionamento
Usar Semantic Versioning (MAJOR.MINOR.PATCH):
- `0.1.0`: Versão inicial com setup, doctor, info
- `0.2.0`: Adição de pkg commands
- `1.0.0`: Release estável

## 6. Dependências

### 6.1 Gerenciamento
- Usar `go get` para adicionar dependências
- Manter `go.mod` e `go.sum` sincronizados
- Revisar licenças de dependências

### 6.2 Dependências Iniciais
- `github.com/spf13/cobra`: CLI framework
- `gopkg.in/yaml.v3`: YAML parsing
- `github.com/fatih/color`: Colorized output (opcional)

## 7. Checklist de Qualidade

Antes de fazer commit:
- [ ] Código formatado (`make fmt`)
- [ ] Linter passou (`make lint`)
- [ ] Testes passaram (`make test`)
- [ ] Build bem-sucedido (`make build`)
- [ ] Cobertura de testes adequada
- [ ] Documentação atualizada
- [ ] Mensagem de commit clara

## 8. Próximos Passos

1. Criar `.golangci.yml` com configuração de linters
2. Criar `Makefile` com automação
3. Implementar `setup`, `doctor`, `info` e `uninstall`
4. Adicionar testes para cada comando
5. Documentar API pública
