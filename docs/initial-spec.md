projeto do cockpit

o cockpit será baseado em:
- um cli (que também poderá ser usado por humandos) que a IA vai usar sempre que precisar executar ou interagir com alguma coisa
- uma base de conhecimento que a IA sempre vai alimentar quando necessário
- agentes de IA, skills e rules que tem como objetivo evoluir o harness

sobre o cli:
- o cli será uma aplicação de commandline extensivel que pode instalar pacotes via cli (cockpit pkg install <nome-do-pacote>)
- o cli deve ser multi-language
- o cli deve ser multiplataforma
- o cli deve ter log para tudo que for executado
- o cli deve ser auditavel
- o cli deve ter um vault nativo (usando o keyrings do SO) para armazenar segredos
- o cli deve ter settings configuravel (onde posso setar o IA que estou usando)
- por mais que o core seja em go (pois assim podemos criar um único binário, os pacotes poderam ser em qualquer em python, node, rust, bash, powershell, etc) ou seja, cada pacote que tenha cli, deve ter um arquivo manifesto que direciona o tipo de executavel a ser usado para executar o comando.

imagino que a estrutura de um pacote seria assim:

packages/
- my-cool-package/
  - cockpit-package.yaml
  - cli/
  - kbs/
    - ...
  - rules/
    - ...
  - skills/
    - ...
  - agents/
    - ...
    
e quando instalarmos o cockpit, todos os arquivos do cockpit (caches, logs, pacotes, etc) serão armazenados na pasta ~/.cockpit. 

comandos que eu imagino:

cockpit setup # configuração inicial do cockpit
  - seleciona a lingua padrão do cockpit (pt-br, en-us ...)
  - cria o vault
  - seleciona a IA (devin cli, antigravity, goose, claude-code)
  - move os agentes e skills e rules e hooks para as pastas globais das IAs selecionadas (precisamos mapear quais são as pastas globais de cada um dos providers )

cockpit doctor # verficar se o cockpit está configurado corretamente

cockpit info # mostrar informações do cockpit
cockpit info <package> # mostrar informações de um pacote

cockpit pkg list # listar pacotes instalados
cockpit pkg search <termo> # buscar pacotes
cockpit pkg install <package> # instalar pacote
cockpit pkg remove <package> # remover pacote
cockpit pkg update <package> # atualizar pacote

cockpit vault # gerenciar segredos do vault

cockpit agents list # listar agentes
cockpit agents search <termo> # buscar agentes
cockpit agents install <agent> # instalar agente
cockpit agents remove <agent> # remover agente
cockpit agents update <agent> # atualizar agente

cockpit rules list # listar rules
cockpit rules search <termo> # buscar rules
cockpit rules install <rule> # instalar rule
cockpit rules remove <rule> # remover rule
cockpit rules update <rule> # atualizar rule

cockpit skills list # listar skills
cockpit skills search <termo> # buscar skills
cockpit skills install <skill> # instalar skill
cockpit skills remove <skill> # remover skill
cockpit skills update <skill> # atualizar skill

cockpit hooks list # listar hooks
cockpit hooks search <termo> # buscar hooks
cockpit hooks install <hook> # instalar hook
cockpit hooks remove <hook> # remover hook
cockpit hooks update <hook> # atualizar hook

cockpit kb list # listar kbs
cockpit kb search <termo> # buscar kbs
cockpit kb install <kb> # instalar kb
cockpit kb remove <kb> # remover kb
cockpit kb update <kb> # atualizar kb

sobre a base de conhecimento (kb) inicialmente será um script que busca em pastas (com find e grep ou outro comando mais eficiente que consiga pesquisar de forma eficiente em arquivos e pastas de formas recursivas)