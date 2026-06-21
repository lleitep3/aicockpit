# Devin Permissions

O Devin executa em ambientes sandboxed ou no computador local do usuário dependendo do tipo de interação. Quando executado localmente via CLI ou Windsurf, o sistema emprega arquivos de configuração para liberar ou bloquear ativamente a execução de comandos shell e acesso a diretórios.

**Referência Oficial:** [Devin Documentation](https://docs.devin.ai/)

## Diretórios e Arquivos de Permissões
- **Global:** `~/.devin/config.local.json`
- **Project-level:** `.devin/config.local.json`

## Formato do Arquivo
Arquivo JSON contendo uma chave `"permissions"` e subchaves para listas de aprovação (`allow`).

## Padrão e Schema
```json
{
  "permissions": {
    "allow": [
      "Exec(rtk)",
      "Exec(make)",
      "Exec(git)"
    ]
  }
}
```

## Boas Práticas
1. **Segurança por Defeito (Default Deny):** O Devin tende a bloquear a execução de ferramentas não reconhecidas, pedindo aprovação explícita do usuário. Use este arquivo para permitir antecipadamente (`grant`) comandos que fazem parte do seu workflow canônico (ex: binários internos, linters, `rtk`).
2. **Separação de Escopo:** Comandos puramente específicos a um projeto devem ficar em `.devin/config.local.json` do workspace. Deixe o `~/.devin/config.local.json` estritamente para ferramentas globais (como o `cockpit` e o `rtk`).
