#!/bin/bash
# scripts/setup-homebrew-tap.sh - Sets up Homebrew tap for mitl
set -euo pipefail

echo "🍺 Setting up Homebrew tap for mitl..."

TAP_REPO="mitl-cli/homebrew-tap"

if command -v gh &> /dev/null; then
    echo "Creating tap repository (if it doesn't exist)..."
    gh repo create "$TAP_REPO" --public --description "Homebrew tap for mitl" || true
fi

TAP_DIR="homebrew-tap"
if [ ! -d "$TAP_DIR" ]; then
    git clone "git@github.com:${TAP_REPO}.git" "$TAP_DIR" 2>/dev/null || mkdir -p "$TAP_DIR"
fi

cd "$TAP_DIR"

mkdir -p Formula
cp ../HomebrewFormula/mitl.rb Formula/

cat > README.md << 'EOF'
# mitl Homebrew Tap

## Installation

```bash
brew tap mitl-cli/tap
brew install mitl
```

## Upgrade

```bash
brew upgrade mitl
```

## Uninstall

```bash
brew uninstall mitl
brew untap mitl-cli/tap
```

EOF

git add .
git commit -m "Add mitl formula v0.1.0-alpha" || true
git push origin main || true

echo "✅ Homebrew tap ready!"
echo ""
echo "Users can now install with:"
echo "  brew tap mitl-cli/tap"
echo "  brew install mitl"

