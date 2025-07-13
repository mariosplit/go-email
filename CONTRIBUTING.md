# Contributing to go-email

First off, thank you for considering contributing to go-email! It's people like you that make go-email such a great tool.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When you create a bug report, please include as many details as possible using our issue template.

**Great Bug Reports** tend to have:
- A quick summary and/or background
- Steps to reproduce (be specific!)
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:
- A clear and descriptive title
- A detailed description of the proposed enhancement
- Examples of how the enhancement would be used
- Why this enhancement would be useful to most users

### Pull Requests

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code follows the existing style
6. Issue that pull request!

## Development Process

1. **Fork and Clone**
   ```bash
   git clone https://github.com/YOUR-USERNAME/go-email.git
   cd go-email
   ```

2. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Changes**
   - Write your code
   - Add tests
   - Update documentation

4. **Test Your Changes**
   ```bash
   make test
   make lint
   ```

5. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: add amazing feature"
   ```

   We use conventional commits:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `test:` for test additions/changes
   - `refactor:` for code refactoring
   - `chore:` for maintenance tasks

6. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

## Style Guide

### Go Code Style

- Follow standard Go conventions
- Use `gofmt` to format your code
- Use meaningful variable names
- Add comments for exported functions
- Keep functions small and focused

### Documentation Style

- Use clear, concise language
- Include code examples where helpful
- Keep lines under 80 characters when possible
- Use proper markdown formatting

### Example Code Style

```go
// SendEmail sends an email using the configured provider.
// It returns an error if the send operation fails.
func (c *Client) SendEmail(ctx context.Context, msg *Message) error {
    // Validate message first
    if err := msg.Validate(); err != nil {
        return fmt.Errorf("invalid message: %w", err)
    }
    
    // Send using provider
    return c.provider.Send(ctx, msg)
}
```

## Testing

- Write unit tests for new functionality
- Maintain or improve code coverage
- Test edge cases and error conditions
- Use table-driven tests where appropriate

Example test:

```go
func TestMessageValidation(t *testing.T) {
    tests := []struct {
        name    string
        message *Message
        wantErr bool
    }{
        {
            name: "valid message",
            message: &Message{
                From:    "test@example.com",
                To:      []string{"recipient@example.com"},
                Subject: "Test",
                Body:    "Test body",
            },
            wantErr: false,
        },
        {
            name: "missing from",
            message: &Message{
                To:      []string{"recipient@example.com"},
                Subject: "Test",
                Body:    "Test body",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.message.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Adding a New Provider

To add support for a new email provider:

1. Create a new file: `providers/yourprovider/yourprovider.go`
2. Implement the `Provider` interface
3. Add configuration struct to `config.go`
4. Update `NewClient` in `email.go`
5. Add documentation and examples
6. Add provider-specific tests

## Questions?

Feel free to open an issue with your question or reach out to the maintainers.

Thank you for contributing! 🎉
