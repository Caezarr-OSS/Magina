package internal

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"encoding/base64"

	"golang.org/x/term"
)

// Credentials represents authentication credentials for a registry
type Credentials struct {
	Username string
	Password string
	Auth     string // Base64 encoded string of "username:password"
}

// AuthHandler handles authentication for registries
type AuthHandler struct {
	configs map[string]*Credentials
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		configs: make(map[string]*Credentials),
	}
}

// GetCredentials returns the credentials for a registry
func (h *AuthHandler) GetCredentials(registryURL string) (*Credentials, error) {
	// Check if we already have the credentials cached
	if creds, ok := h.configs[registryURL]; ok {
		return creds, nil
	}

	// Try to retrieve from environment variables
	creds, err := h.getCredsFromEnv(registryURL)
	if err == nil {
		h.configs[registryURL] = creds
		return creds, nil
	}

	// Prompt the user for credentials
	creds, err = h.promptCredentials(registryURL)
	if err != nil {
		return nil, err
	}

	// Cache the credentials
	h.configs[registryURL] = creds
	return creds, nil
}

// getCredsFromEnv attempts to retrieve credentials from environment variables
func (h *AuthHandler) getCredsFromEnv(registryURL string) (*Credentials, error) {
	// Clean the URL to create a valid prefix for environment variables
	prefix := strings.NewReplacer(
		"https://", "",
		"http://", "",
		".", "_",
		"/", "_",
		"-", "_",
	).Replace(strings.ToUpper(registryURL))

	// Look for environment variables
	username := os.Getenv(prefix + "_USERNAME")
	password := os.Getenv(prefix + "_PASSWORD")

	if username == "" || password == "" {
		return nil, fmt.Errorf("credentials not found in environment")
	}

	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	return &Credentials{
		Username: username,
		Password: password,
		Auth:     auth,
	}, nil
}

// promptCredentials prompts the user for credentials
func (h *AuthHandler) promptCredentials(registryURL string) (*Credentials, error) {
	fmt.Printf("Authentication required for %s\n", registryURL)
	
	fmt.Print("Username: ")
	var username string
	fmt.Scanln(&username)

	fmt.Print("Password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // New line after password

	auth := base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(username) + ":" + strings.TrimSpace(string(password))))

	return &Credentials{
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(string(password)),
		Auth:     auth,
	}, nil
}
