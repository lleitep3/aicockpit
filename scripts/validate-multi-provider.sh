#!/bin/bash

# AICockpit Multi-Provider Installation Validator
# Validates installation across all AI providers (Devin, Goose, Claude Code, GitHub Copilot)

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
REQUIRED_DIRS=("agents" "skills" "hooks" "kb" "logs" "cache")

# Required files
REQUIRED_FILES=("config.yaml" "manifest.yaml")

# Statistics
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

# Functions
print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED_CHECKS++))
}

print_error() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED_CHECKS++))
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

check_provider_workspace() {
    local provider=$1
    local path=${PROVIDER_PATHS[$provider]}
    
    print_header "Checking $provider workspace"
    
    if [ ! -d "$path" ]; then
        print_error "$provider workspace not found at $path"
        return 1
    fi
    
    print_success "$provider workspace exists at $path"
    ((TOTAL_CHECKS++))
    
    return 0
}

check_provider_directories() {
    local provider=$1
    local path=${PROVIDER_PATHS[$provider]}
    
    print_header "Checking $provider directories"
    
    for dir in "${REQUIRED_DIRS[@]}"; do
        ((TOTAL_CHECKS++))
        if [ -d "$path/$dir" ]; then
            print_success "Directory $dir exists"
        else
            print_warning "Directory $dir not found (will be created on first use)"
        fi
    done
}

check_provider_files() {
    local provider=$1
    local path=${PROVIDER_PATHS[$provider]}
    
    print_header "Checking $provider files"
    
    for file in "${REQUIRED_FILES[@]}"; do
        ((TOTAL_CHECKS++))
        if [ -f "$path/$file" ]; then
            print_success "File $file exists"
        else
            print_error "File $file not found"
        fi
    done
}

check_provider_components() {
    local provider=$1
    local path=${PROVIDER_PATHS[$provider]}
    
    print_header "Checking $provider components"
    
    # Check agents
    ((TOTAL_CHECKS++))
    if [ -d "$path/agents" ]; then
        local agent_count=$(find "$path/agents" -maxdepth 1 -type d | wc -l)
        if [ $agent_count -gt 1 ]; then
            print_success "Found $((agent_count - 1)) agents"
        else
            print_warning "No agents installed"
        fi
    fi
    
    # Check skills
    ((TOTAL_CHECKS++))
    if [ -d "$path/skills" ]; then
        local skill_count=$(find "$path/skills" -maxdepth 1 -type d | wc -l)
        if [ $skill_count -gt 1 ]; then
            print_success "Found $((skill_count - 1)) skills"
        else
            print_warning "No skills installed"
        fi
    fi
    
    # Check hooks
    ((TOTAL_CHECKS++))
    if [ -d "$path/hooks" ]; then
        local hook_count=$(find "$path/hooks" -maxdepth 1 -type d | wc -l)
        if [ $hook_count -gt 1 ]; then
            print_success "Found $((hook_count - 1)) hooks"
        else
            print_warning "No hooks installed"
        fi
    fi
    
    # Check KB documents
    ((TOTAL_CHECKS++))
    if [ -d "$path/kb" ]; then
        local doc_count=$(find "$path/kb" -name "*.md" | wc -l)
        if [ $doc_count -gt 0 ]; then
            print_success "Found $doc_count KB documents"
        else
            print_warning "No KB documents found"
        fi
    fi
}

check_manifest_integrity() {
    local provider=$1
    local path=${PROVIDER_PATHS[$provider]}
    
    print_header "Checking $provider manifest integrity"
    
    ((TOTAL_CHECKS++))
    if [ -f "$path/manifest.yaml" ]; then
        # Check if manifest is valid YAML
        if grep -q "^version:" "$path/manifest.yaml"; then
            print_success "Manifest has valid structure"
        else
            print_error "Manifest structure is invalid"
        fi
    else
        print_error "Manifest file not found"
    fi
}

check_config_integrity() {
    local provider=$1
    local path=${PROVIDER_PATHS[$provider]}
    
    print_header "Checking $provider configuration integrity"
    
    ((TOTAL_CHECKS++))
    if [ -f "$path/config.yaml" ]; then
        # Check if config is valid YAML
        if grep -q "^version:" "$path/config.yaml"; then
            print_success "Configuration has valid structure"
        else
            print_error "Configuration structure is invalid"
        fi
        
        # Check if ai_provider is set correctly
        ((TOTAL_CHECKS++))
        if grep -q "ai_provider: \"$provider\"" "$path/config.yaml"; then
            print_success "ai_provider correctly set to $provider"
        else
            print_warning "ai_provider not set to $provider"
        fi
    else
        print_error "Configuration file not found"
    fi
}

check_permissions() {
    local provider=$1
    local path=${PROVIDER_PATHS[$provider]}
    
    print_header "Checking $provider permissions"
    
    ((TOTAL_CHECKS++))
    if [ -d "$path" ]; then
        local perms=$(stat -c %a "$path" 2>/dev/null || stat -f %A "$path" 2>/dev/null)
        if [ "$perms" = "700" ] || [ "$perms" = "755" ]; then
            print_success "Workspace permissions are secure ($perms)"
        else
            print_warning "Workspace permissions may need adjustment ($perms)"
        fi
    fi
}

check_disk_space() {
    local provider=$1
    local path=${PROVIDER_PATHS[$provider]}
    
    print_header "Checking $provider disk space"
    
    ((TOTAL_CHECKS++))
    if [ -d "$path" ]; then
        local size=$(du -sh "$path" 2>/dev/null | cut -f1)
        print_info "Workspace size: $size"
    fi
}

compare_providers() {
    print_header "Comparing installations across providers"
    
    # Check if all providers have the same agents
    echo -e "\n${BLUE}Agents:${NC}"
    for provider in "${PROVIDERS[@]}"; do
        local path=${PROVIDER_PATHS[$provider]}
        if [ -d "$path/agents" ]; then
            local agents=$(ls -d "$path/agents"/*/ 2>/dev/null | xargs -n1 basename | tr '\n' ',' | sed 's/,$//')
            if [ -z "$agents" ]; then
                agents="(none)"
            fi
            echo "  $provider: $agents"
        else
            echo "  $provider: (workspace not found)"
        fi
    done
    
    # Check if all providers have the same skills
    echo -e "\n${BLUE}Skills:${NC}"
    for provider in "${PROVIDERS[@]}"; do
        local path=${PROVIDER_PATHS[$provider]}
        if [ -d "$path/skills" ]; then
            local skills=$(ls -d "$path/skills"/*/ 2>/dev/null | xargs -n1 basename | tr '\n' ',' | sed 's/,$//')
            if [ -z "$skills" ]; then
                skills="(none)"
            fi
            echo "  $provider: $skills"
        else
            echo "  $provider: (workspace not found)"
        fi
    done
    
    # Check if all providers have the same hooks
    echo -e "\n${BLUE}Hooks:${NC}"
    for provider in "${PROVIDERS[@]}"; do
        local path=${PROVIDER_PATHS[$provider]}
        if [ -d "$path/hooks" ]; then
            local hooks=$(ls -d "$path/hooks"/*/ 2>/dev/null | xargs -n1 basename | tr '\n' ',' | sed 's/,$//')
            if [ -z "$hooks" ]; then
                hooks="(none)"
            fi
            echo "  $provider: $hooks"
        else
            echo "  $provider: (workspace not found)"
        fi
    done
}

generate_report() {
    print_header "Validation Report"
    
    echo -e "${BLUE}Summary:${NC}"
    echo "  Total Checks: $TOTAL_CHECKS"
    echo -e "  ${GREEN}Passed: $PASSED_CHECKS${NC}"
    echo -e "  ${RED}Failed: $FAILED_CHECKS${NC}"
    
    if [ $FAILED_CHECKS -eq 0 ]; then
        echo -e "\n${GREEN}✓ All validations passed!${NC}"
        return 0
    else
        echo -e "\n${RED}✗ Some validations failed. Please review the output above.${NC}"
        return 1
    fi
}

# Main execution
main() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║  AICockpit Multi-Provider Installation Validator           ║"
    echo "║  Validates installation across all AI providers            ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    # Check each provider
    for provider in "${PROVIDERS[@]}"; do
        if check_provider_workspace "$provider"; then
            check_provider_directories "$provider"
            check_provider_files "$provider"
            check_provider_components "$provider"
            check_manifest_integrity "$provider"
            check_config_integrity "$provider"
            check_permissions "$provider"
            check_disk_space "$provider"
        fi
    done
    
    # Compare across providers
    compare_providers
    
    # Generate report
    generate_report
}

# Run main function
main
