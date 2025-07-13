# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-01-13

### Added
- Initial release of go-email package
- Support for Outlook 365 via Microsoft Graph API
- Support for Gmail via Gmail API
- OAuth2 authentication for both providers
- HTML email support
- File attachments with automatic MIME type detection
- CC and BCC recipients
- Environment variable configuration
- Context support for timeouts and cancellation
- Comprehensive error handling
- QuickSend convenience function
- Version information API

### Security
- Secure OAuth2 authentication flow
- No hardcoded credentials
- Support for service account authentication (Gmail)

### Documentation
- Comprehensive README with examples
- Integration guide for production use
- Provider-specific setup guides
- API documentation
- Multiple working examples

### Testing
- Unit tests for core functionality
- Integration test framework
- Mock implementations for testing

## [Unreleased]

### Planned
- SendGrid provider support
- AWS SES provider support
- Email template engine
- Webhook support for email events
- Batch sending optimization
- Email validation utilities
- Retry mechanism with exponential backoff
- Connection pooling for high-volume sending
