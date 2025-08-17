# mitl Quick Start Guide

## 5-Minute Setup

### 1. Install mitl

```bash
# Using Homebrew (recommended)
brew tap mitl-cli/tap
brew install mitl

# Or quick install
curl -fsSL https://mitl.run/install.sh | bash
```

### 2. Verify Installation

```bash
mitl doctor
```

You should see:

```
✅ Container Runtime: Using Apple Container (optimal)
✅ Cache: Ready
✅ Disk Space: 45.2GB free
✅ Permissions: All correct

Performance Score: 95/100 (A+)
```

### 3. Try It Out

Navigate to any project and run:

```bash
# Laravel project
cd my-laravel-app
mitl run php artisan serve

# Node.js project
cd my-node-app
mitl run npm test

# Python project
cd my-python-app
mitl run python manage.py runserver
```

## Common Workflows

### Laravel Development

```bash
# First time setup
mitl run composer install
mitl run php artisan key:generate
mitl run php artisan migrate

# Daily development
mitl run php artisan serve
mitl run php artisan test
mitl run npm run dev
```

### Node.js Development

```bash
# Install dependencies (auto-converts to pnpm)
mitl run npm install

# Run scripts
mitl run npm test
mitl run npm run build
mitl run npm start
```

### Interactive Shell

```bash
# Open a shell in the container
mitl shell

# Now you're inside the container
composer install
php artisan tinker
exit
```

## Performance Tips

### 1. Install Apple Container (Apple Silicon only)

If you have an M1/M2/M3 Mac:

```bash
# Check if you have it
mitl doctor

# If not, download from:
# https://developer.apple.com/virtualization
```

### 2. Use pnpm for Node.js

mitl automatically converts npm/yarn to pnpm, saving 70% disk space:

```bash
# This automatically uses pnpm
mitl run npm install
```

### 3. Clean Old Capsules

```bash
# See what's cached
mitl cache list

# Clean old ones
mitl cache clean
```

## Debugging

### Verbose Mode

See what mitl is doing:

```bash
mitl run -v npm test
```

### Debug Mode

Full debug output with timing:

```bash
mitl run --debug npm test
```

### Check Logs

Debug logs are saved to:

```bash
~/.mitl/logs/mitl-YYYY-MM-DD.log
```

## Next Steps

- Read the full [README](../README.md)
- Check [Troubleshooting](../README.md#troubleshooting)
- Report issues at [GitHub Issues](https://github.com/mitl-cli/mitl/issues)

