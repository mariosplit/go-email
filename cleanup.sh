#!/bin/bash
# cleanup.sh - Clean up go-email project before Git push

echo "🧹 Cleaning up go-email project..."

# Remove all .bat files (Windows batch scripts)
echo "Removing batch files..."
find . -name "*.bat" -type f -delete 2>/dev/null || true

# Remove temporary and backup files
echo "Removing temporary files..."
find . -name "*.tmp" -type f -delete 2>/dev/null || true
find . -name "*.bak" -type f -delete 2>/dev/null || true
find . -name "*~" -type f -delete 2>/dev/null || true

# Remove old example files that are not in organized directories
echo "Cleaning examples directory..."
cd examples 2>/dev/null || exit 1
rm -f check-*.go 2>/dev/null || true
rm -f gmail-*.go 2>/dev/null || true
rm -f test-*.go 2>/dev/null || true
rm -f simple*.go 2>/dev/null || true
rm -f universal-sender.go 2>/dev/null || true
rm -f invoice_example.go 2>/dev/null || true
cd ..

# Remove duplicate documentation files
echo "Cleaning documentation..."
rm -f GMAIL-*.md 2>/dev/null || true
rm -f GOOGLE-*.md 2>/dev/null || true
rm -f SERVICE-*.md 2>/dev/null || true
rm -f NO-BIAS-*.md 2>/dev/null || true
rm -f SLOW-*.md 2>/dev/null || true
rm -f YES-*.md 2>/dev/null || true
rm -f FIX-*.md 2>/dev/null || true
rm -f NETWORK-*.md 2>/dev/null || true
rm -f SOLUTION-*.md 2>/dev/null || true
rm -f SEND-*.md 2>/dev/null || true
rm -f PUBLIC-*.md 2>/dev/null || true
rm -f STATUS.md 2>/dev/null || true
rm -f GIT-*.md 2>/dev/null || true

# Keep only essential markdown files
echo "Keeping essential documentation..."
# Files to keep: README.md, LICENSE, CHANGELOG.md, CONTRIBUTING.md, API.md, INTEGRATION.md, PRE-PUBLISH-CHECKLIST.md, PROJECT-SUMMARY.md

# Remove PowerShell scripts (keep only essential ones)
echo "Cleaning scripts..."
rm -f copy-dependencies.ps1 2>/dev/null || true
rm -f quick-fix.ps1 2>/dev/null || true

# Remove shell scripts except cleanup
rm -f init-git.sh 2>/dev/null || true
rm -f setup.sh 2>/dev/null || true

# Format Go code
echo "Formatting Go code..."
go fmt ./... 2>/dev/null || true

# Tidy dependencies
echo "Tidying Go modules..."
go mod tidy 2>/dev/null || true

# Run tests to ensure nothing is broken
echo "Running tests..."
go test ./... 2>/dev/null || echo "⚠️  Some tests failed - please check"

# Final file count
echo ""
echo "📊 Project statistics:"
echo "Go files: $(find . -name "*.go" -type f | wc -l)"
echo "Markdown files: $(find . -name "*.md" -type f | wc -l)"
echo "Total files: $(find . -type f -not -path "./.git/*" | wc -l)"

echo ""
echo "✅ Cleanup complete!"
echo ""
echo "Next steps:"
echo "1. Review the changes: git status"
echo "2. Add files: git add ."
echo "3. Commit: git commit -m 'Initial commit: go-email v1.0.0'"
echo "4. Add remote: git remote add origin https://github.com/YOUR-USERNAME/go-email.git"
echo "5. Push: git push -u origin main"
