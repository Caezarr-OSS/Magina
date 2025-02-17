package internal

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"encoding/base64"

	"golang.org/x/term"
)

// Credentials représente les informations d'authentification pour un registry
type Credentials struct {
	Username string
	Password string
	Auth     string // Base64 encoded string of "username:password"
}

// AuthHandler gère l'authentification aux registries
type AuthHandler struct {
	configs map[string]*Credentials
}

// NewAuthHandler crée un nouveau gestionnaire d'authentification
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		configs: make(map[string]*Credentials),
	}
}

// GetCredentials retourne les credentials pour un registry
func (h *AuthHandler) GetCredentials(registryURL string) (*Credentials, error) {
	// Vérifier si on a déjà les credentials en cache
	if creds, ok := h.configs[registryURL]; ok {
		return creds, nil
	}

	// Essayer de récupérer depuis les variables d'environnement
	creds, err := h.getCredsFromEnv(registryURL)
	if err == nil {
		h.configs[registryURL] = creds
		return creds, nil
	}

	// Demander les credentials à l'utilisateur
	creds, err = h.promptCredentials(registryURL)
	if err != nil {
		return nil, err
	}

	// Sauvegarder en cache
	h.configs[registryURL] = creds
	return creds, nil
}

// getCredsFromEnv tente de récupérer les credentials depuis les variables d'environnement
func (h *AuthHandler) getCredsFromEnv(registryURL string) (*Credentials, error) {
	// Nettoyer l'URL pour créer un préfixe valide pour les variables d'environnement
	prefix := strings.NewReplacer(
		"https://", "",
		"http://", "",
		".", "_",
		"/", "_",
		"-", "_",
	).Replace(strings.ToUpper(registryURL))

	// Chercher les variables d'environnement
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

// promptCredentials demande les credentials à l'utilisateur
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
	fmt.Println() // Nouvelle ligne après le mot de passe

	auth := base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(username) + ":" + strings.TrimSpace(string(password))))

	return &Credentials{
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(string(password)),
		Auth:     auth,
	}, nil
}
