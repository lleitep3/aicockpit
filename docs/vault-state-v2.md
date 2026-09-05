# Vault lock state v2

O estado de bloqueio do vault é um envelope autenticado com AES-GCM. A chave
de 32 bytes é aleatória e fica no keyring do sistema, no serviço interno
`aicockpit-internal`; ela não é adicionada ao índice público de credenciais.
`version` e `key_id` fazem parte do AAD, e o nonce aleatório é gerenciado pelo
AEAD da biblioteca padrão.

## Modelo de ameaça

O arquivo de estado pode ser lido ou adulterado por alguém que não possui a
entrada correspondente no keyring. Nessa situação, corrupção, chave ausente,
versão desconhecida e alteração de permissões falham fechando o acesso. A
proteção não impede um processo que já executa como o mesmo usuário e consegue
usar o keyring, nem oferece proteção contra rollback de um ciphertext v2
autêntico antigo.

## Migração

Estados v1 não são desbloqueados nem migrados automaticamente, porque a chave
v1 era derivável de dados públicos. Execute:

```sh
cockpit vault migrate-state --confirm
```

A operação cria um backup exclusivo dos bytes v1, gera uma nova chave, grava
um estado v2 bloqueado e descarta todos os grants legados. Credenciais, índice
público, permissões de pacotes e senha mestra não são alterados. Sem `--confirm`
o comando pede `MIGRATE`; em automação sem stdin ele retorna erro. Se o estado
já for v2 válido, a operação é no-op. Um estado corrompido ou de versão
desconhecida exige recuperação explícita e não é tratado como v1.

Perder a chave do keyring também bloqueia o acesso; a leitura nunca cria uma
chave substituta silenciosamente. A recuperação deve preservar credenciais e
reinicializar somente o estado de grants, conforme procedimento operacional do
ambiente.

## Operação e testes

Status e autorização falham fechando quando não conseguem autenticar o estado.
As mutações fazem read-modify-write sob lock entre processos e publicam a nova
imagem em memória somente após escrita atômica, `sync`, fechamento e rename.
Testes usam keyring falso, diretórios temporários e não executam migração,
lock/unlock ou reset no vault pessoal.

O verificador legado da senha mestra (SHA-256 e cifra dependente do sistema) é
uma dívida separada: corrupção agora é erro, não “senha desabilitada”, mas a
modernização completa do verificador não faz parte desta entrega.
