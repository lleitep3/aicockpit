## Descrição / Description
<!-- Descreva de forma clara e concisa o que foi feito neste Pull Request e qual o problema ele resolve. -->
<!-- Describe clearly and concisely what was done in this Pull Request and what problem it solves. -->

## Tipo de Mudança / Type of Change
<!-- Marque a opção apropriada com um 'x' / Check the appropriate option with an 'x' -->
- [ ] 🐛 Bug fix (mudança não-quebrante que corrige um problema / non-breaking change which fixes an issue)
- [ ] ✨ Nova Feature (mudança não-quebrante que adiciona funcionalidade / non-breaking change which adds functionality)
- [ ] 💥 Breaking change (mudança quebrante / fix or feature that would cause existing functionality to not work as expected)
- [ ] 📝 Documentação (apenas mudanças em documentação / documentation changes only)
- [ ] ♻️ Refatoração (mudança no código que não corrige bug nem adiciona feature / code change that neither fixes a bug nor adds a feature)
- [ ] 🔧 Configuração/CI (mudanças nos arquivos de configuração ou pipeline / changes to configuration files or pipeline)

## Impacto na Versão (Semantic Versioning)
<!-- Com base na mudança acima, como isso afetará a versão do sistema? -->
- [ ] **PATCH** (Correções / Fixes)
- [ ] **MINOR** (Novas features compatíveis com versões anteriores / Backward-compatible new features)
- [ ] **MAJOR** (Mudanças que quebram a compatibilidade / Breaking changes)

## Evidências / Evidence
<!-- Insira aqui prints (screenshots), gifs, ou o output do console demonstrando o antes e o depois, ou a funcionalidade em ação. -->
<!-- Add screenshots, gifs, or console outputs here demonstrating the before and after, or the feature in action. -->

<details>
<summary>Clique para expandir as evidências</summary>

```
Cole seu log ou adicione a imagem aqui.
```

</details>

## Comandos para Teste / Test Commands
<!-- Quais comandos o revisor deve rodar para testar manualmente a sua feature? Forneça exemplos claros. -->
<!-- What commands should the reviewer run to manually test your feature? Provide clear examples. -->

```bash
# Exemplo:
go run main.go kb search "termo"
```

## Checklist de Qualidade / Quality Checklist
- [ ] O código passou em todos os testes locais (`make check` / `go test ./...`)
- [ ] Foram adicionados/atualizados testes unitários cobrindo as mudanças (mínimo de 90% de coverage)
- [ ] A documentação (`docs/` ou `README.md`) foi atualizada (se aplicável)
- [ ] Os commits estão no formato Conventional Commits (ex: `feat(scope): description`)
- [ ] Não há warnings no *linter* (`golangci-lint`)
