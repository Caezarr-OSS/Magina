package internal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Session represents a session of the application
// that maintains credentials in memory
type Session struct {
	credentials map[string]*Credentials
	mu          sync.RWMutex
}

// NewSession creates a new session
func NewSession() *Session {
	return &Session{
		credentials: make(map[string]*Credentials),
	}
}

// GetCredentials retrieves the credentials for a registry
// If the credentials do not exist, asks the user
func (s *Session) GetCredentials(registryURL string) (*Credentials, error) {
	s.mu.RLock()
	creds, exists := s.credentials[registryURL]
	s.mu.RUnlock()

	if exists {
		return creds, nil
	}

	// Ask the user for credentials
	creds, err := s.promptCredentials(registryURL)
	if err != nil {
		return nil, err
	}

	// Store the credentials in memory
	s.mu.Lock()
	s.credentials[registryURL] = creds
	s.mu.Unlock()

	return creds, nil
}

// promptCredentials asks the user for credentials
func (s *Session) promptCredentials(registryURL string) (*Credentials, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Authentication required for %s\n", registryURL)
	
	fmt.Print("Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)

	fmt.Print("Password: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}
	password = strings.TrimSpace(password)

	return &Credentials{
		Username: username,
		Password: password,
	}, nil
}

// Clear clears all credentials from the session
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Clear the credentials map
	s.credentials = make(map[string]*Credentials)
}
