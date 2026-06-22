---
name: kb-search
description: Search the local Knowledge Base (KB) for documentation, references, and guides using AICockpit.
---

# Habilidade: kb-search

Esta habilidade permite que você (a IA) busque documentações, guias de melhores práticas, resoluções de problemas e referências técnicas armazenadas na Base de Conhecimento local do desenvolvedor.

## Quando usar

*   Antes de iniciar qualquer tarefa de desenvolvimento para entender o contexto do projeto.
*   Ao encontrar erros de compilação, testes ou linter para buscar soluções na seção de troubleshooting.
*   Sempre que precisar de especificações de estilo de código, fluxo de trabalho Git ou arquitetura.

## Como usar

Execute o seguinte comando de CLI no terminal para realizar uma busca por termos específicos:

```bash
cockpit kb search "seu termo de busca"
```

### Exemplo de Uso:

Se o desenvolvedor solicitar para depurar as configurações de logs:
1. Execute: `cockpit kb search "logging"`
2. Analise os resultados retornados.
3. Se necessário, abra e leia o arquivo retornado utilizando a ferramenta de leitura de arquivos adequada.
