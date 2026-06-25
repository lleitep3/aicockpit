#!/bin/bash

# Exemplo de como usar o vault AICockpit via CLI em shell scripts
# Demonstrando o uso de namespace para isolamento entre pacotes

set -e

NAMESPACE="example-app"

echo "=== Exemplo de Uso do Vault AICockpit via CLI ==="
echo ""
echo "Namespace: $NAMESPACE"
echo ""

# 1. Criar secrets de teste
echo "1. Criando secrets de teste..."
cockpit vault set --namespace $NAMESPACE example_api_key --value "sk-test-1234567890abcdef"
echo "   ✓ example_api_key criado"

cockpit vault set --namespace $NAMESPACE example_db_password --value "super_secret_password_123"
echo "   ✓ example_db_password criado"

echo ""

# 2. Recuperar e usar secrets
echo "2. Recuperando secrets do vault..."

API_KEY=$(cockpit vault get --namespace $NAMESPACE example_api_key)
DB_PASSWORD=$(cockpit vault get --namespace $NAMESPACE example_db_password)

echo "   ✓ API Key recuperada: ${API_KEY:0:10}...${API_KEY: -4}"
echo "   ✓ DB Password recuperada: ${DB_PASSWORD:0:4}...${DB_PASSWORD: -4}"

echo ""

# 3. Usar em comando simulado
echo "3. Usando secrets em comandos..."

# Simular conexão com banco de dados
echo "   Conectando ao banco de dados..."
echo "   psql -h localhost -U user -d mydb -P password=\"${DB_PASSWORD:0:4}...\""
echo "   ✓ Conexão simulada com sucesso"

# Simular chamada de API
echo "   Chamando API..."
echo "   curl -H \"Authorization: Bearer ${API_KEY:0:10}...\" https://api.example.com/endpoint"
echo "   ✓ Chamada simulada com sucesso"

echo ""

# 4. Exemplo de uso em variáveis de ambiente
echo "4. Exportando secrets para variáveis de ambiente..."
export EXAMPLE_API_KEY=$(cockpit vault get --namespace $NAMESPACE example_api_key)
export EXAMPLE_DB_PASSWORD=$(cockpit vault get --namespace $NAMESPACE example_db_password)

echo "   ✓ EXAMPLE_API_KEY=${EXAMPLE_API_KEY:0:10}..."
echo "   ✓ EXAMPLE_DB_PASSWORD=${EXAMPLE_DB_PASSWORD:0:4}..."

echo ""

# 5. Exemplo de script de configuração
echo "5. Gerando arquivo de configuração..."
cat > /tmp/example_config.json <<EOF
{
  "api": {
    "key": "${EXAMPLE_API_KEY}",
    "endpoint": "https://api.example.com"
  },
  "database": {
    "password": "${EXAMPLE_DB_PASSWORD}",
    "host": "localhost"
  }
}
EOF

echo "   ✓ Arquivo de configuração gerado: /tmp/example_config.json"
echo "   Conteúdo (com secrets mascarados):"
cat /tmp/example_config.json | sed "s/${EXAMPLE_API_KEY}/${EXAMPLE_API_KEY:0:10}.../g" | sed "s/${EXAMPLE_DB_PASSWORD}/${EXAMPLE_DB_PASSWORD:0:4}.../g"

echo ""

# 6. Teste de isolamento de namespace
echo "6. Testando isolamento de namespace..."
echo "   Tentando acessar secret de outro namespace..."
OTHER_KEY=$(cockpit vault get --namespace other-app example_api_key 2>&1 || echo "NOT_FOUND")
if [[ "$OTHER_KEY" == *"secret not found"* ]] || [[ "$OTHER_KEY" == *"NOT_FOUND"* ]]; then
    echo "   ✓ Acesso cross-namespace BLOQUEADO (isolamento funcionando)"
else
    echo "   ✗ AVISO: Acesso cross-namespace permitido"
fi

echo ""

# 7. Limpar secrets de teste
echo "7. Limpando secrets de teste..."
cockpit vault remove --namespace $NAMESPACE example_api_key
echo "   ✓ example_api_key removido"

cockpit vault remove --namespace $NAMESPACE example_db_password
echo "   ✓ example_db_password removido"

rm -f /tmp/example_config.json
echo "   ✓ Arquivo de configuração removido"

echo ""
echo "=== Benefícios do Namespace ==="
echo "✓ Isolamento entre pacotes/applicações"
echo "✓ Previne acesso acidental a secrets de outros pacotes"
echo "✓ Compatível com lock/unlock (namespace bypassa lock)"
echo "✓ Segurança melhorada com isolamento automático"
echo ""
echo "=== Exemplo concluído ==="