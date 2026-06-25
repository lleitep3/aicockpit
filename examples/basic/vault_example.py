#!/usr/bin/env python3
"""
Exemplo de como usar o vault AICockpit em Python via CLI
"""

import subprocess
import json
import os

def get_secret(key: str) -> str:
    """
    Recupera um secret do vault AICockpit
    
    Args:
        key: Nome do secret a recuperar
        
    Returns:
        O valor do secret
        
    Raises:
        Exception: Se houver erro ao recuperar o secret
    """
    try:
        result = subprocess.run(
            ['cockpit', 'vault', 'get', key],
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        raise Exception(f"Erro ao recuperar secret '{key}': {e.stderr}")
    except FileNotFoundError:
        raise Exception("cockpit command não encontrado. Verifique se o AICockpit está instalado.")

def mask_secret(secret: str, visible_chars: int = 4) -> str:
    """Mascara um secret para exibição segura"""
    if len(secret) <= visible_chars * 2:
        return "***"
    return secret[:visible_chars] + "..." + secret[-visible_chars:]

def main():
    print("=== Exemplo de Uso do Vault AICockpit em Python ===\n")
    
    # 1. Criar secrets de teste
    print("1. Criando secrets de teste...")
    
    test_secrets = {
        "python_api_key": "sk-python-test-1234567890",
        "python_db_url": "postgresql://user:pass@localhost/db",
        "python_secret": "my_super_secret_value_123"
    }
    
    for key, value in test_secrets.items():
        try:
            subprocess.run(
                ['cockpit', 'vault', 'set', key, '--value', value],
                capture_output=True,
                check=True
            )
            print(f"   ✓ {key} criado")
        except subprocess.CalledProcessError as e:
            print(f"   ✗ Erro ao criar {key}: {e}")
    
    print()
    
    # 2. Recuperar secrets
    print("2. Recuperando secrets do vault...")
    
    try:
        api_key = get_secret("python_api_key")
        print(f"   ✓ API Key: {mask_secret(api_key)}")
    except Exception as e:
        print(f"   ✗ Erro ao recuperar API key: {e}")
    
    try:
        db_url = get_secret("python_db_url")
        print(f"   ✓ Database URL: {mask_secret(db_url, 8)}")
    except Exception as e:
        print(f"   ✗ Erro ao recuperar database URL: {e}")
    
    try:
        secret = get_secret("python_secret")
        print(f"   ✓ Secret: {mask_secret(secret)}")
    except Exception as e:
        print(f"   ✗ Erro ao recuperar secret: {e}")
    
    print()
    
    # 3. Usar em simulação de chamada de API
    print("3. Simulando chamada de API com Python...")
    
    try:
        api_key = get_secret("python_api_key")
        
        # Simular chamada de API (em produção, seria requests.get/post)
        print(f"   Fazendo requisição para https://api.example.com/endpoint")
        print(f"   Headers: Authorization: Bearer {mask_secret(api_key)}")
        print(f"   ✓ Resposta simulada: {{'status': 'success', 'data': [...]}}")
        
    except Exception as e:
        print(f"   ✗ Erro na chamada de API: {e}")
    
    print()
    
    # 4. Gerar configuração
    print("4. Gerando arquivo de configuração...")
    
    try:
        config = {
            "api": {
                "key": get_secret("python_api_key"),
                "endpoint": "https://api.example.com"
            },
            "database": {
                "url": get_secret("python_db_url")
            },
            "secrets": {
                "app_secret": get_secret("python_secret")
            }
        }
        
        config_path = "/tmp/python_vault_config.json"
        with open(config_path, 'w') as f:
            json.dump(config, f, indent=2)
        
        print(f"   ✓ Configuração gerada: {config_path}")
        
        # Mostrar conteúdo com secrets mascarados
        with open(config_path, 'r') as f:
            content = f.read()
            for key, value in test_secrets.items():
                content = content.replace(value, mask_secret(value, 8))
            print("   Conteúdo (com secrets mascarados):")
            for line in content.split('\n'):
                print(f"   {line}")
                
    except Exception as e:
        print(f"   ✗ Erro ao gerar configuração: {e}")
    
    print()
    
    # 5. Limpar secrets de teste
    print("5. Limpando secrets de teste...")
    
    for key in test_secrets.keys():
        try:
            subprocess.run(
                ['cockpit', 'vault', 'remove', key],
                capture_output=True,
                check=True
            )
            print(f"   ✓ {key} removido")
        except subprocess.CalledProcessError as e:
            print(f"   ✗ Erro ao remover {key}: {e}")
    
    # Remover arquivo de configuração
    try:
        os.remove("/tmp/python_vault_config.json")
        print("   ✓ Arquivo de configuração removido")
    except FileNotFoundError:
        pass
    
    print()
    print("=== Exemplo concluído ===")

if __name__ == "__main__":
    main()