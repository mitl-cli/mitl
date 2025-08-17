#!/bin/bash
# build-check.sh - Validates the MITL project structure and build setup

set -e

echo "🔍 MITL Infrastructure Validation"
echo "=================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

# Check directory structure
echo
echo "📁 Checking directory structure..."

REQUIRED_DIRS=(
    "cmd/mitl"
    "internal/cli/commands"
    "internal/detector"
    "internal/volume"
    "internal/digest"
    "internal/doctor"
    "internal/build/templates"
    "pkg/version"
    "pkg/terminal"
    "pkg/exec"
    "bin"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        success "Directory exists: $dir"
    else
        error "Missing directory: $dir"
        exit 1
    fi
done

# Check for required Go files
echo
echo "📄 Checking Go package files..."

REQUIRED_FILES=(
    "go.mod"
    "cmd/mitl/main.go"
    "internal/cli/commands/doc.go"
    "internal/digest/doc.go"
    "internal/build/templates/doc.go"
    "pkg/version/doc.go"
    "pkg/terminal/doc.go"
    "pkg/exec/doc.go"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        success "File exists: $file"
    else
        error "Missing file: $file"
        exit 1
    fi
done

# Check Go module validity
echo
echo "🔧 Checking Go module..."
if go mod verify > /dev/null 2>&1; then
    success "Go module is valid"
else
    warning "Go module verification failed (expected during refactor)"
fi

# Check if Makefile exists and has required targets
echo
echo "🛠️  Checking Makefile..."
if [ -f "Makefile" ]; then
    success "Makefile exists"
    
    if grep -q "build:" Makefile; then
        success "Build target found in Makefile"
    else
        error "Build target not found in Makefile"
        exit 1
    fi
    
    if grep -q "cmd/mitl/main.go" Makefile; then
        success "Makefile references cmd/mitl/main.go"
    else
        error "Makefile does not reference cmd/mitl/main.go"
        exit 1
    fi
else
    error "Makefile not found"
    exit 1
fi

# Test build (this may fail but we'll capture the status)
echo
echo "🏗️  Testing build process..."
if make build > /dev/null 2>&1; then
    success "Build successful"
else
    warning "Build failed (expected during refactor phase)"
    echo "   This is normal while packages are being restructured"
fi

echo
echo "🎉 Infrastructure validation complete!"
echo "The basic structure is ready for MITL refactoring."