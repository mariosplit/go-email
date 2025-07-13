// auth.go - Authentication helpers for OAuth2 providers
package email

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// GmailAuthHelper provides utilities for Gmail OAuth2 authentication.
// It handles the OAuth2 flow for obtaining access tokens for Gmail API.
type GmailAuthHelper struct {
	// CredentialsJSON contains the OAuth2 client credentials from Google Cloud Console
	CredentialsJSON []byte
}

// NewGmailAuthHelper creates a new Gmail authentication helper with the provided credentials.
//
// The credentials should be the JSON file downloaded from Google Cloud Console
// when creating OAuth2 credentials for a desktop application.
//
// Example:
//
//	creds, err := os.ReadFile("credentials.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	helper := email.NewGmailAuthHelper(creds)
func NewGmailAuthHelper(credentialsJSON []byte) *GmailAuthHelper {
	return &GmailAuthHelper{
		CredentialsJSON: credentialsJSON,
	}
}

// Authenticate performs the OAuth2 authentication flow and returns the access token as JSON.
// This method will prompt the user to visit a URL and enter an authorization code.
//
// The returned token can be saved and reused for future email sending without
// requiring re-authentication.
//
// Example:
//
//	helper := email.NewGmailAuthHelper(credentialsJSON)
//	token, err := helper.Authenticate()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Save token for future use
//	err = os.WriteFile("token.json", token, 0600)
func (g *GmailAuthHelper) Authenticate() ([]byte, error) {
	config, err := google.ConfigFromJSON(g.CredentialsJSON, gmail.GmailSendScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	token, err := g.getTokenFromWeb(config)
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %w", err)
	}

	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal token: %w", err)
	}

	return tokenJSON, nil
}

// getTokenFromWeb uses the OAuth2 flow to get a token from the web.
// It prints the auth URL to stdout and waits for the user to enter the authorization code.
func (g *GmailAuthHelper) getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser:\n%v\n\n", authURL)
	fmt.Print("Enter the authorization code: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

// AuthenticateGmailFromFile is a convenience function that reads credentials from a file
// and performs the OAuth2 authentication flow.
//
// This is useful for one-time authentication setup.
//
// Example:
//
//	token, err := email.AuthenticateGmailFromFile("credentials.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Save the token for future use
//	err = os.WriteFile("token.json", token, 0600)
//	if err != nil {
//	    log.Fatal(err)
//	}
func AuthenticateGmailFromFile(credentialsFile string) ([]byte, error) {
	creds, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %w", err)
	}

	helper := NewGmailAuthHelper(creds)
	return helper.Authenticate()
}
