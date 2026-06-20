#!/bin/bash

# Script to generate command documentation from Go code
# This script extracts command metadata and generates markdown documentation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DOCS_DIR="$PROJECT_ROOT/docs/commands"
CMD_DIR="$PROJECT_ROOT/cmd"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}AICockpit Command Documentation Generator${NC}"
echo "=========================================="
echo ""

# Check if we're in the right directory
if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
    echo -e "${RED}Error: go.mod not found. Are you in the project root?${NC}"
    exit 1
fi

# Check if docs/commands directory exists
if [ ! -d "$DOCS_DIR" ]; then
    echo -e "${YELLOW}Creating docs/commands directory...${NC}"
    mkdir -p "$DOCS_DIR"
fi

echo -e "${YELLOW}Scanning command files...${NC}"

# Find all command files
COMMANDS=$(find "$CMD_DIR" -name "*.go" -type f | grep -v test | grep -v root.go)

if [ -z "$COMMANDS" ]; then
    echo -e "${RED}No command files found${NC}"
    exit 1
fi

echo -e "${GREEN}Found commands:${NC}"

# Extract command names and generate documentation
for cmd_file in $COMMANDS; do
    cmd_name=$(basename "$cmd_file" .go)
    
    # Skip if it's not a command file (doesn't have NewXCommand function)
    if ! grep -q "func New${cmd_name^}Command" "$cmd_file" 2>/dev/null; then
        continue
    fi
    
    echo -e "  ${BLUE}•${NC} $cmd_name"
    
    # Check if documentation already exists
    doc_file="$DOCS_DIR/${cmd_name}.md"
    if [ -f "$doc_file" ]; then
        echo -e "    ${GREEN}✓${NC} Documentation exists"
    else
        echo -e "    ${YELLOW}⚠${NC} Documentation missing (create manually)"
    fi
done

echo ""
echo -e "${YELLOW}Validating documentation...${NC}"

# Check for missing documentation
MISSING=0
for cmd_file in $COMMANDS; do
    cmd_name=$(basename "$cmd_file" .go)
    
    if ! grep -q "func New${cmd_name^}Command" "$cmd_file" 2>/dev/null; then
        continue
    fi
    
    doc_file="$DOCS_DIR/${cmd_name}.md"
    if [ ! -f "$doc_file" ]; then
        echo -e "  ${RED}✗${NC} Missing: $cmd_name"
        MISSING=$((MISSING + 1))
    fi
done

if [ $MISSING -gt 0 ]; then
    echo -e "${YELLOW}Found $MISSING missing documentation file(s)${NC}"
    echo -e "${YELLOW}Please create documentation for:${NC}"
    for cmd_file in $COMMANDS; do
        cmd_name=$(basename "$cmd_file" .go)
        if ! grep -q "func New${cmd_name^}Command" "$cmd_file" 2>/dev/null; then
            continue
        fi
        doc_file="$DOCS_DIR/${cmd_name}.md"
        if [ ! -f "$doc_file" ]; then
            echo -e "  ${BLUE}•${NC} docs/commands/${cmd_name}.md"
        fi
    done
else
    echo -e "${GREEN}✓ All commands have documentation${NC}"
fi

echo ""
echo -e "${YELLOW}Checking documentation completeness...${NC}"

# Check for required sections in documentation
INCOMPLETE=0
for doc_file in "$DOCS_DIR"/*.md; do
    if [ "$(basename "$doc_file")" = "README.md" ] || [ "$(basename "$doc_file")" = "COMMAND_TEMPLATE.md" ]; then
        continue
    fi
    
    cmd_name=$(basename "$doc_file" .md)
    
    # Check for required sections
    REQUIRED_SECTIONS=("Overview" "Usage" "Description" "Examples" "Exit Codes")
    
    for section in "${REQUIRED_SECTIONS[@]}"; do
        if ! grep -q "^## $section" "$doc_file"; then
            echo -e "  ${RED}✗${NC} Missing section '$section' in $cmd_name.md"
            INCOMPLETE=$((INCOMPLETE + 1))
        fi
    done
done

if [ $INCOMPLETE -gt 0 ]; then
    echo -e "${YELLOW}Found $INCOMPLETE incomplete documentation section(s)${NC}"
else
    echo -e "${GREEN}✓ All documentation sections are complete${NC}"
fi

echo ""
echo -e "${YELLOW}Generating command index...${NC}"

# Count documented commands
DOC_COUNT=$(find "$DOCS_DIR" -name "*.md" -type f | grep -v README.md | grep -v COMMAND_TEMPLATE.md | wc -l)

echo -e "${GREEN}✓ Documentation generated for $DOC_COUNT commands${NC}"

echo ""
echo -e "${YELLOW}Summary:${NC}"
echo "  Documentation directory: $DOCS_DIR"
echo "  Documented commands: $DOC_COUNT"
echo "  Missing documentation: $MISSING"
echo "  Incomplete sections: $INCOMPLETE"

echo ""
if [ $MISSING -eq 0 ] && [ $INCOMPLETE -eq 0 ]; then
    echo -e "${GREEN}✅ All command documentation is complete!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some documentation needs attention${NC}"
    exit 1
fi
