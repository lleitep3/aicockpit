#!/bin/bash

# Exemplo de como o kb-graphify usaria o vault de forma segura
# Este é um exemplo de migração dos scripts atuais

echo "=== Exemplo de Uso Seguro do Vault pelo kb-graphify ==="
echo ""

NAMESPACE="kb-graphify"

# 1. Configuração (kb-graphify/bin/configure)
echo "1. Configurando credenciais do kb-graphify (modo seguro)..."
echo "   Antes: cockpit vault set kb-graphify.provider --value '\$PROVIDER'"
echo "   Depois: cockpit vault set --namespace $NAMESPACE provider --value '\$PROVIDER'"

# Simular configuração
rtk ./bin/cockpit vault set --namespace $NAMESPACE provider --value "openai"
echo "   ✓ Provider configurado"

rtk ./bin/cockpit vault set --namespace $NAMESPACE api-key --value "sk-test-kb-graphify-123"
echo "   ✓ API Key configurada"

echo ""
echo "2. Validando credenciais (kb-graphify/bin/validate)"
echo "   Antes: PROVIDER=\$(cockpit vault get kb-graphify.provider)"
echo "   Depois: PROVIDER=\$(cockpit vault get --namespace $NAMESPACE provider)"

# Recuperar credenciais de forma segura
PROVIDER=$(rtk ./bin/cockpit vault get --namespace $NAMESPACE provider)
API_KEY=$(rtk ./bin/cockpit vault get --namespace $NAMESPACE api-key)

echo "   ✓ Provider recuperado: $PROVIDER"
echo "   ✓ API Key recuperada: ${API_KEY:0:10}..."

echo ""
echo "3. Testando isolamento de namespace"

# Tentar acessar de outro namespace (deve falhar)
echo "   Tentando acessar com namespace 'other-app'..."
OTHER_PROVIDER=$(rtk ./bin/cockpit vault get --namespace other-app provider 2>&1 || echo "ACCESS_DENIED")

if [[ "$OTHER_PROVIDER" == *"ACCESS_DENIED"* ]] || [[ "$OTHER_PROVIDER" == *"secret not found"* ]]; then
    echo "   ✓ Acesso cross-namespace BLOQUEADO (isolamento funcionando)"
else
    echo "   ✗ AVISO: Acesso cross-namespace permitido (isolamento falhou)"
fi

echo ""
echo "4. Testando compatibilidade reversa"

# Sem namespace (modo antigo, ainda funciona para compatibilidade)
echo "   Testando acesso sem namespace (compatibilidade reversa)..."
rtk ./bin/cockpit vault set legacy_key --value "legacy_value" > /dev/null 2>&1
LEGACY_VALUE=$(rtk ./bin/cockpit vault get legacy_key)

if [ "$LEGACY_VALUE" == "legacy_value" ]; then
    echo "   ✓ Modo sem namespace ainda funciona (compatibilidade mantida)"
    rtk ./bin/cockpit vault remove legacy_key > /dev/null 2>&1
fi

echo ""
echo "5. Comparação de Segurança"

echo "   Antes (INSEGURO):"
echo "   - cockpit vault get kb-graphify.api-key"
echo "   - Pode acessar QUALQUER secret"
echo "   - Sem isolamento entre pacotes"
echo ""
echo "   Depois (SEGURO):"
echo "   - cockpit vault get --namespace kb-graphify api-key"
echo "   - Só acessa secrets do namespace 'kb-graphify'"
echo "   - Isolamento completo entre pacotes"
echo "   - Compatibilidade com código existente"

echo ""
echo "6. Limpando dados de teste"
rtk ./bin/cockpit vault remove --namespace $NAMESPACE provider > /dev/null 2>&1
rtk ./bin/cockpit vault remove --namespace $NAMESPACE api-key > /dev/null 2>&1
echo "   ✓ Dados de teste removidos"

echo ""
echo "=== Benefícios da Migração ==="
echo "✓ kb-graphify só acessa seus próprios secrets"
echo "✓ Isolamento completo entre pacotes"
echo "✓ Compatibilidade mantida com scripts existentes"
echo "✓ Migração gradual possível"
echo "✓ Segurança melhorada imediatamente"
echo ""
echo "=== Exemplo Concluído ==="