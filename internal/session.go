package internal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Session représente une session de l'application
// qui maintient les credentials en mémoire
type Session struct {
	credentials map[string]*Credentials
	mu          sync.RWMutex
}

// NewSession crée une nouvelle session
func NewSession() *Session {
	return &Session{
		credentials: make(map[string]*Credentials),
	}
}

// GetCredentials récupère les credentials pour un registry
// Si les credentials n'existent pas, demande à l'utilisateur
func (s *Session) GetCredentials(registryURL string) (*Credentials, error) {
	s.mu.RLock()
	creds, exists := s.credentials[registryURL]
	s.mu.RUnlock()

	if exists {
		return creds, nil
	}

	// Demander les credentials à l'utilisateur
	creds, err := s.promptCredentials(registryURL)
	if err != nil {
		return nil, err
	}

	// Stocker les credentials en mémoire
	s.mu.Lock()
	s.credentials[registryURL] = creds
	s.mu.Unlock()

	return creds, nil
}

// promptCredentials demande les credentials à l'utilisateur
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

// Clear efface tous les credentials de la session
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Effacer la map des credentials
	s.credentials = make(map[string]*Credentials)
}
