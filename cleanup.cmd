@echo off
REM cleanup.cmd - Clean up go-email project before Git push (Windows version)

echo Cleaning up go-email project...

REM Remove all .bat files except this one
echo Removing batch files...
for %%f in (*.bat) do (
    if not "%%f"=="cleanup.cmd" del "%%f" 2>nul
)

REM Remove temporary files
echo Removing temporary files...
del /s *.tmp 2>nul
del /s *.bak 2>nul
del /s *.log 2>nul

REM Clean examples directory
echo Cleaning examples directory...
cd examples 2>nul
if not errorlevel 1 (
    del check-*.go 2>nul
    del gmail-*.go 2>nul
    del test-*.go 2>nul
    del simple*.go 2>nul
    del universal-sender.go 2>nul
    del invoice_example.go 2>nul
    cd ..
)

REM Remove duplicate documentation
echo Cleaning documentation...
del GMAIL-*.md 2>nul
del GOOGLE-*.md 2>nul
del SERVICE-*.md 2>nul
del NO-BIAS-*.md 2>nul
del SLOW-*.md 2>nul
del YES-*.md 2>nul
del FIX-*.md 2>nul
del NETWORK-*.md 2>nul
del SOLUTION-*.md 2>nul
del SEND-*.md 2>nul
del PUBLIC-*.md 2>nul
del STATUS.md 2>nul
del GIT-*.md 2>nul
del QUICKSTART.md 2>nul
del PUBLISH-GUIDE.md 2>nul

REM Remove PowerShell scripts
echo Cleaning scripts...
del copy-dependencies.ps1 2>nul
del quick-fix.ps1 2>nul
del create-legal-mamagement-dir-structure.ps1 2>nul
del create-repo.ps1 2>nul

REM Remove shell scripts
del init-git.sh 2>nul
del setup.sh 2>nul

REM Remove other config files
del config.minimal.go 2>nul
del gmail-smtp.go 2>nul
del gmail-workspace.go 2>nul
del standalone-test.go 2>nul
del admin-email-template.txt 2>nul
del service-account-info.txt 2>nul

REM Format Go code
echo Formatting Go code...
go fmt ./...

REM Tidy dependencies
echo Tidying Go modules...
go mod tidy

REM Run tests
echo Running tests...
go test ./...
if errorlevel 1 (
    echo WARNING: Some tests failed - please check
)

echo.
echo Cleanup complete!
echo.
echo Essential files kept:
echo - Source files: *.go
echo - Documentation: README.md, CHANGELOG.md, CONTRIBUTING.md, API.md, INTEGRATION.md
echo - Configuration: go.mod, go.sum, .gitignore, Makefile
echo - GitHub: .github/
echo - Examples: examples/
echo - Docs: docs/
echo.
echo Next steps:
echo 1. Review changes: git status
echo 2. Initialize git: git init
echo 3. Add files: git add .
echo 4. Commit: git commit -m "Initial commit: go-email v1.0.0"
echo 5. Add remote: git remote add origin https://github.com/YOUR-USERNAME/go-email.git
echo 6. Push: git push -u origin main
echo.
pause
