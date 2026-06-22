# Antigravity Permissions

O modelo de segurança do Google Antigravity exige concessões (grants) antecipadas para que o agente possa agir sem pedir permissão humana interativa a todo o momento. As permissões incluem acesso a arquivos (leitura e gravação), execução de comandos de terminal, conexões de rede e execução fora da sandbox.

**Referência Oficial:** Uso interno (Google Antigravity SDK).

## Diretórios e Arquivos de Permissões
- **Global:** `~/.gemini/config/config.json`

## Formato do Arquivo
Arquivo JSON de configuração contendo a chave `grantedPermissions`.

## Padrão e Schema
```json
{
  "grantedPermissions": [
    {
      "Action": "command",
      "Target": "rtk"
    },
    {
      "Action": "command",
      "Target": "make"
    },
    {
      "Action": "write_file",
      "Target": "/home/lleite/projects"
    }
  ]
}
```

## Boas Práticas
1. **Request the Narrowest Scope:** O cockpit só deve injetar no `config.json` o escopo mais estrito necessário. Nunca dê wildcard (`*`) ou permissões de root. Se o agente precisa executar `rtk`, declare `"Target": "rtk"`.
2. **Separação de Rede e Execução:** Comandos como `curl`, `wget` ou instâncias de `pip`/`npm` frequentemente não devem ser colocadas de forma persistente no arquivo, obrigando que o usuário revise o download ou execução manual interativamente. Utilize as grants do cockpit apenas para binários internos da stack segura.
