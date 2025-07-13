package email

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Mock provider for testing
type mockProvider struct {
	sendFunc func(ctx context.Context, msg *Message) error
	calls    []Message
}

func (m *mockProvider) Send(ctx context.Context, msg *Message) error {
	m.calls = append(m.calls, *msg)
	if m.sendFunc != nil {
		return m.sendFunc(ctx, msg)
	}
	return nil
}

func TestMessageValidation(t *testing.T) {
	tests := []struct {
		name    string
		message *Message
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			message: &Message{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
				Body:    "Test body content",
			},
			wantErr: false,
		},
		{
			name: "valid message with cc and bcc",
			message: &Message{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Cc:      []string{"cc@example.com"},
				Bcc:     []string{"bcc@example.com"},
				Subject: "Test Subject",
				Body:    "Test body content",
			},
			wantErr: false,
		},
		{
			name: "valid html message with attachment",
			message: &Message{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
				Body:    "<h1>HTML Content</h1>",
				HTML:    true,
				Attachments: []Attachment{
					{
						Filename: "test.txt",
						Content:  []byte("test content"),
						MimeType: "text/plain",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing from address",
			message: &Message{
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
				Body:    "Test body",
			},
			wantErr: true,
			errMsg:  "from address is required",
		},
		{
			name: "missing recipients",
			message: &Message{
				From:    "sender@example.com",
				To:      []string{},
				Subject: "Test Subject",
				Body:    "Test body",
			},
			wantErr: true,
			errMsg:  "at least one recipient is required",
		},
		{
			name: "nil recipients",
			message: &Message{
				From:    "sender@example.com",
				Subject: "Test Subject",
				Body:    "Test body",
			},
			wantErr: true,
			errMsg:  "at least one recipient is required",
		},
		{
			name: "missing subject",
			message: &Message{
				From: "sender@example.com",
				To:   []string{"recipient@example.com"},
				Body: "Test body",
			},
			wantErr: true,
			errMsg:  "subject is required",
		},
		{
			name: "missing body",
			message: &Message{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Subject",
			},
			wantErr: true,
			errMsg:  "body is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "unsupported provider",
			config: &Config{
				Provider: "unsupported",
			},
			wantErr: true,
			errMsg:  "unsupported provider: unsupported",
		},
		{
			name: "outlook without config",
			config: &Config{
				Provider: "outlook365",
			},
			wantErr: true,
			errMsg:  "outlook configuration is required",
		},
		{
			name: "gmail without config",
			config: &Config{
				Provider: "gmail",
			},
			wantErr: true,
			errMsg:  "gmail configuration is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("NewClient() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestClientSend(t *testing.T) {
	tests := []struct {
		name     string
		message  *Message
		sendErr  error
		wantErr  bool
		checkCtx bool
	}{
		{
			name: "successful send",
			message: &Message{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test",
				Body:    "Test body",
			},
			sendErr: nil,
			wantErr: false,
		},
		{
			name: "provider error",
			message: &Message{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test",
				Body:    "Test body",
			},
			sendErr: errors.New("provider error"),
			wantErr: true,
		},
		{
			name: "invalid message",
			message: &Message{
				From: "sender@example.com",
				// Missing required fields
			},
			wantErr: true,
		},
		{
			name: "context timeout",
			message: &Message{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test",
				Body:    "Test body",
			},
			sendErr:  context.DeadlineExceeded,
			wantErr:  true,
			checkCtx: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProvider{
				sendFunc: func(ctx context.Context, msg *Message) error {
					if tt.checkCtx {
						// Simulate delay to trigger timeout
						select {
						case <-ctx.Done():
							return ctx.Err()
						case <-time.After(100 * time.Millisecond):
							return nil
						}
					}
					return tt.sendErr
				},
			}

			client := &Client{provider: mock}

			var err error
			if tt.checkCtx {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()
				err = client.SendWithContext(ctx, tt.message)
			} else {
				err = client.Send(tt.message)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(mock.calls) != 1 {
				t.Errorf("Expected 1 call to provider, got %d", len(mock.calls))
			}
		})
	}
}

func TestQuickSend(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		creds    interface{}
		from     string
		to       string
		subject  string
		body     string
		wantErr  bool
	}{
		{
			name:     "invalid provider",
			provider: "invalid",
			creds:    &OutlookConfig{},
			from:     "from@example.com",
			to:       "to@example.com",
			subject:  "Test",
			body:     "Body",
			wantErr:  true,
		},
		{
			name:     "outlook with wrong creds type",
			provider: "outlook365",
			creds:    &GmailConfig{},
			from:     "from@example.com",
			to:       "to@example.com",
			subject:  "Test",
			body:     "Body",
			wantErr:  true,
		},
		{
			name:     "gmail with wrong creds type",
			provider: "gmail",
			creds:    &OutlookConfig{},
			from:     "from@example.com",
			to:       "to@example.com",
			subject:  "Test",
			body:     "Body",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := QuickSend(tt.provider, tt.creds, tt.from, tt.to, tt.subject, tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("QuickSend() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	if version != Version {
		t.Errorf("GetVersion() = %v, want %v", version, Version)
	}
}

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()

	if info.Version != GetVersion() {
		t.Errorf("VersionInfo.Version = %v, want %v", info.Version, GetVersion())
	}

	if info.Major != VersionMajor {
		t.Errorf("VersionInfo.Major = %v, want %v", info.Major, VersionMajor)
	}

	if info.Minor != VersionMinor {
		t.Errorf("VersionInfo.Minor = %v, want %v", info.Minor, VersionMinor)
	}

	if info.Patch != VersionPatch {
		t.Errorf("VersionInfo.Patch = %v, want %v", info.Patch, VersionPatch)
	}
}

// Benchmark tests
func BenchmarkMessageValidation(b *testing.B) {
	msg := &Message{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test body content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = msg.Validate()
	}
}

func BenchmarkClientSend(b *testing.B) {
	mock := &mockProvider{
		sendFunc: func(ctx context.Context, msg *Message) error {
			return nil
		},
	}

	client := &Client{provider: mock}
	msg := &Message{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test body content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Send(msg)
	}
}
