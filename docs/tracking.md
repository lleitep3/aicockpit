# Tracking de Pacotes

> [!NOTE]
> **Fase de Desenvolvimento:** Implementado no pacote `internal/tracking`. A API é usada internamente pelo sistema de pacotes e pode ser exposta via CLI em versões futuras.

O módulo `tracking` adiciona rastros de proveniência nos arquivos de um pacote Cockpit. Cada arquivo sincronizado para o workspace pode receber um cabeçalho (`header`) descrevendo de qual pacote ele veio, qual versão, e quando foi criado ou modificado.

## Propósito

Facilitar a auditoria e a origem de conteúdo gerado por pacotes. Quando um `skill`, `rule`, `agent` ou `workflow` é instalado, o tracking header permite identificar imediatamente:

- **Pacote** de origem (`package:<name>`)
- **Versão** instalada (`version:<version>`)
- **Data de criação** dos metadados (`created:<timestamp>`)
- **Última modificação** registrada (`updated:<timestamp>`)

## Como Funciona

O módulo expõe duas funções principais em `internal/tracking/header.go`:

### `GenerateHeader`

Gera a string do cabeçalho a partir de um `*packages.Package`:

```go
package := &packages.Package{
    Name:    "meu-pacote",
    Version: "1.0.0",
    Metadata: packages.Metadata{
        CreationDate: "2026-01-01T00:00:00Z",
        LastModified: "2026-01-02T00:00:00Z",
    },
}

header := tracking.GenerateHeader(package)
// // package:meu-pacote version:1.0.0 created:2026-01-01T00:00:00Z updated:2026-01-02T00:00:00Z
```

### `InjectHeader`

Lê um arquivo existente e insere o cabeçalho como primeira linha, preservando o conteúdo original:

```go
err := tracking.InjectHeader("skills/hello/SKILL.md", package)
if err != nil {
    return err
}
```

Após a execução, o arquivo terá o seguinte formato:

```text
// package:meu-pacote version:1.0.0 created:2026-01-01T00:00:00Z updated:2026-01-02T00:00:00Z
# Hello World

Conteúdo original do skill...
```

## Campos dos Metadados

| Campo         | Descrição                                          | Origem                           |
|---------------|----------------------------------------------------|----------------------------------|
| `package`     | Nome canônico do pacote                            | `pkg.Name`                       |
| `version`     | Versão semântica do pacote                         | `pkg.Version`                    |
| `created`     | Data de criação do pacote                          | `pkg.Metadata.CreationDate`      |
| `updated`     | Data da última modificação dos metadados           | `pkg.Metadata.LastModified`      |

Os campos `CreationDate` e `LastModified` são preenchidos automaticamente por `packages.SavePackage` quando o pacote é persistido.

## Eventos Mapeados

O tracking captura os seguintes eventos do ciclo de vida do pacote:

1. **Geração de cabeçalho** (`GenerateHeader`) — disparado sempre que é necessário montar o metadado textual de um pacote.
2. **Injeção de cabeçalho** (`InjectHeader`) — executado durante o sync de assets de um pacote para o workspace, garantindo proveniência nos arquivos copiados.
3. **Leitura de arquivo** — `InjectHeader` verifica a existência e legibilidade do arquivo alvo.
4. **Escrita de arquivo** — o conteúdo original é lido, o cabeçalho é prefixado e o arquivo é reescrito atomicamente.

## Cobertura de Testes

O pacote `internal/tracking` possui testes unitários cobrindo:

- Geração correta do cabeçalho (`TestGenerateHeader`)
- Inserção em arquivo existente (`TestInjectHeader`)
- Falha ao ler arquivo inexistente (`TestInjectHeader_ReadFileError`)
- Falha ao escrever em arquivo somente leitura (`TestInjectHeader_WriteFileError`)

Cobertura atual: 100% das statements.

## Evoluções Futuras

- **Comando CLI** `cockpit tracking inject <arquivo> --package <pacote>` para injeção manual.
- **Formatos configuráveis** de cabeçalho (YAML front-matter, JSON, comentário `//` ou `#`).
- **Tracking de proveniência estendida**, incluindo hash do conteúdo e assinatura do pacote.
- **Integração com hooks** de instalação para ativar/desativar a inserção automática.
- **Métricas de rastreamento** (quantos arquivos possuem header, quais pacotes são mais instalados).
