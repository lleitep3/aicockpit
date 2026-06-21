#!/bin/bash

# AICockpit Multi-Provider Setup Script
# Installs AICockpit for multiple AI providers (Devin, Goose, Claude Code, GitHub Copilot)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Providers
PROVIDERS=("devin" "goose" "claude-code" "github-copilot")

# Provider workspace mappings
declare -A PROVIDER_PATHS=(
    [devin]="$HOME/.cockpit"
    [goose]="$HOME/.goose"
    [claude-code]="$HOME/.claude-code"
    [github-copilot]="$HOME/.github-copilot"
)

# Required directories
REQUIRED_DIRS=("agents" "skills" "hooks" "kb/guides" "kb/examples" "kb/troubleshooting" "logs" "cache" "backups")

# Statistics
TOTAL_PROVIDERS=0
INSTALLED_PROVIDERS=0
FAILED_PROVIDERS=0

# Functions
print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

confirm() {
    local prompt="$1"
    local response
    
    read -p "$(echo -e ${BLUE}$prompt${NC}) (y/n) " response
    [[ "$response" =~ ^[Yy]$ ]]
}

setup_provider() {
    local provider=$1
    local path=${PROVIDER_PATHS[$provider]}
    
    print_header "Setting up $provider"
    
    ((TOTAL_PROVIDERS++))
    
    # Create workspace directory
    if [ ! -d "$path" ]; then
        print_info "Creating workspace at $path"
        mkdir -p "$path"
        chmod 700 "$path"
        print_success "Workspace created"
    else
        print_info "Workspace already exists at $path"
    fi
    
    # Create required directories
    print_info "Creating directory structure"
    for dir in "${REQUIRED_DIRS[@]}"; do
        mkdir -p "$path/$dir"
    done
    print_success "Directory structure created"
    
    # Create config.yaml if it doesn't exist
    if [ ! -f "$path/config.yaml" ]; then
        print_info "Creating configuration file"
        cat > "$path/config.yaml" << EOF
version: "0.2.0"
language: "en-us"
log_level: "info"
ai_provider: "$provider"

kb:
  roots:
    - $path/kb

agents:
  enabled: true
  
skills:
  enabled: true
  
hooks:
  enabled: true
EOF
        print_success "Configuration file created"
    else
        print_info "Configuration file already exists"
    fi
    
    # Create manifest.yaml if it doesn't exist
    if [ ! -f "$path/manifest.yaml" ]; then
        print_info "Creating installation manifest"
        cat > "$path/manifest.yaml" << EOF
version: "1.0"
cockpit_version: "0.2.0"
installed_at: "$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
last_updated: "$(date -u +'%Y-%m-%dT%H:%M:%SZ')"

agents: []
skills: []
hooks: []
modules: []

config:
  backup_dir: "$path/backups"
  keep_backup: true
  log_operations: true
EOF
        print_success "Installation manifest created"
    else
        print_info "Installation manifest already exists"
    fi
    
    # Copy KB documents
    print_info "Copying knowledge base documents"
    local kb_source="$(dirname "$0")/../ai-assets/knowledge-base"
    if [ -d "$kb_source" ]; then
        cp -r "$kb_source"/* "$path/kb/" 2>/dev/null || true
        print_success "Knowledge base documents copied"
    else
        print_warning "Knowledge base source not found"
    fi
    
    # Copy example components
    print_info "Copying example components"
    local examples_source="$(dirname "$0")/../ai-assets/examples"
    if [ -d "$examples_source" ]; then
        # Copy agents
        if [ -d "$examples_source/agents" ]; then
            cp -r "$examples_source/agents"/* "$path/agents/" 2>/dev/null || true
        fi
        
        # Copy skills
        if [ -d "$examples_source/skills" ]; then
            cp -r "$examples_source/skills"/* "$path/skills/" 2>/dev/null || true
        fi
        
        # Copy hooks
        if [ -d "$examples_source/hooks" ]; then
            cp -r "$examples_source/hooks"/* "$path/hooks/" 2>/dev/null || true
        fi
        
        print_success "Example components copied"
    else
        print_warning "Examples source not found"
    fi
    
    # Set permissions
    chmod 700 "$path"
    chmod 700 "$path"/*
    
    print_success "$provider setup completed successfully"
    ((INSTALLED_PROVIDERS++))
}

verify_provider() {
    local provider=$1
    local path=${PROVIDER_PATHS[$provider]}
    
    print_header "Verifying $provider installation"
    
    local errors=0
    
    # Check workspace
    if [ ! -d "$path" ]; then
        print_error "Workspace not found at $path"
        ((errors++))
    else
        print_success "Workspace exists"
    fi
    
    # Check directories
    for dir in "${REQUIRED_DIRS[@]}"; do
        if [ ! -d "$path/$dir" ]; then
            print_warning "Directory $dir not found"
        else
            print_success "Directory $dir exists"
        fi
    done
    
    # Check files
    if [ ! -f "$path/config.yaml" ]; then
        print_error "Configuration file not found"
        ((errors++))
    else
        print_success "Configuration file exists"
    fi
    
    if [ ! -f "$path/manifest.yaml" ]; then
        print_error "Manifest file not found"
        ((errors++))
    else
        print_success "Manifest file exists"
    fi
    
    # Check components
    local agent_count=$(find "$path/agents" -maxdepth 1 -type d 2>/dev/null | wc -l)
    if [ $agent_count -gt 1 ]; then
        print_success "Found $((agent_count - 1)) agents"
    else
        print_warning "No agents installed"
    fi
    
    local skill_count=$(find "$path/skills" -maxdepth 1 -type d 2>/dev/null | wc -l)
    if [ $skill_count -gt 1 ]; then
        print_success "Found $((skill_count - 1)) skills"
    else
        print_warning "No skills installed"
    fi
    
    local hook_count=$(find "$path/hooks" -maxdepth 1 -type d 2>/dev/null | wc -l)
    if [ $hook_count -gt 1 ]; then
        print_success "Found $((hook_count - 1)) hooks"
    else
        print_warning "No hooks installed"
    fi
    
    if [ $errors -eq 0 ]; then
        print_success "$provider verification passed"
        return 0
    else
        print_error "$provider verification failed with $errors errors"
        return 1
    fi
}

generate_report() {
    print_header "Installation Report"
    
    echo -e "${BLUE}Summary:${NC}"
    echo "  Total Providers: $TOTAL_PROVIDERS"
    echo -e "  ${GREEN}Installed: $INSTALLED_PROVIDERS${NC}"
    echo -e "  ${RED}Failed: $FAILED_PROVIDERS${NC}"
    
    if [ $FAILED_PROVIDERS -eq 0 ]; then
        echo -e "\n${GREEN}✓ All providers installed successfully!${NC}"
        return 0
    else
        echo -e "\n${RED}✗ Some providers failed to install.${NC}"
        return 1
    fi
}

# Main execution
main() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║  AICockpit Multi-Provider Setup                            ║"
    echo "║  Installs AICockpit for all AI providers                   ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    # Show providers
    echo -e "\n${BLUE}Available Providers:${NC}"
    for provider in "${PROVIDERS[@]}"; do
        echo "  • $provider (${PROVIDER_PATHS[$provider]})"
    done
    
    # Confirm installation
    if ! confirm "\nProceed with installation for all providers?"; then
        print_info "Installation cancelled"
        exit 0
    fi
    
    # Setup each provider
    for provider in "${PROVIDERS[@]}"; do
        if setup_provider "$provider"; then
            if verify_provider "$provider"; then
                print_success "$provider setup and verification completed"
            else
                print_error "$provider verification failed"
                ((FAILED_PROVIDERS++))
            fi
        else
            print_error "$provider setup failed"
            ((FAILED_PROVIDERS++))
        fi
    done
    
    # Generate report
    generate_report
    
    # Print next steps
    print_header "Next Steps"
    echo -e "${BLUE}To verify all installations:${NC}"
    echo "  bash $(dirname "$0")/validate-multi-provider.sh"
    echo ""
    echo -e "${BLUE}To use a specific provider:${NC}"
    echo "  export COCKPIT_PROVIDER=devin"
    echo "  cockpit kb list"
    echo ""
    echo -e "${BLUE}To install components in all providers:${NC}"
    echo "  cockpit agent install cockpit-builder --all-providers"
    echo ""
}

# Run main function
main
