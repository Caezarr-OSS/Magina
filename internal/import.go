package internal

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// ImportOptions contient les options pour l'opération d'import
type ImportOptions struct {
	CleanOnError bool
	VerboseLevel int
	Credentials  *Credentials
}

// ImportResult représente le résultat d'un import d'image
type ImportResult struct {
	LocalImage       string
	DestinationImage string
	Error           error
}

// ImportHandler gère l'import des images
type ImportHandler struct {
	ctx     context.Context
	options ImportOptions
	logger  *log.Logger
}

// NewImportHandler crée un nouveau gestionnaire d'import
func NewImportHandler(ctx context.Context, options ImportOptions) *ImportHandler {
	return &ImportHandler{
		ctx:     ctx,
		options: options,
		logger:  log.New(log.Writer(), "[IMPORT] ", log.LstdFlags),
	}
}

// ImportImages importe les images vers le registry de destination
func (h *ImportHandler) ImportImages(block *Block) <-chan ImportResult {
	results := make(chan ImportResult)

	go func() {
		defer close(results)

		// Vérifier que le bloc est valide pour l'import
		if err := h.validateImportBlock(block); err != nil {
			results <- ImportResult{Error: err}
			return
		}

		// Configurer l'authentification
		auth := h.getAuthConfig(block.DestinationRegistry.Host)

		// Traiter chaque mapping d'image
		for _, mapping := range block.ImageMappings {
			// Vérifier si l'image est exclue
			if h.isExcluded(mapping.Destination, block.Exclusions) {
				continue
			}

			// Importer l'image
			result := h.importSingleImage(mapping.Source, mapping.Destination, auth)
			results <- result
		}
	}()

	return results
}

// validateImportBlock vérifie que le bloc est valide pour l'import
func (h *ImportHandler) validateImportBlock(block *Block) error {
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}

	if block.DestinationRegistry.Host == "" {
		return fmt.Errorf("destination registry host cannot be empty")
	}

	if len(block.ImageMappings) == 0 {
		return fmt.Errorf("no image mappings found")
	}

	return nil
}

// getAuthConfig configure l'authentification pour le registry
func (h *ImportHandler) getAuthConfig(registryURL string) authn.Authenticator {
	if h.options.Credentials == nil {
		return authn.Anonymous
	}

	return authn.FromConfig(authn.AuthConfig{
		Username: h.options.Credentials.Username,
		Password: h.options.Credentials.Password,
		Auth:     h.options.Credentials.Auth,
	})
}

// isExcluded vérifie si une image est dans la liste des exclusions
func (h *ImportHandler) isExcluded(image string, exclusions []string) bool {
	for _, exclusion := range exclusions {
		if strings.Contains(image, exclusion) {
			return true
		}
	}
	return false
}

// importSingleImage importe une seule image
func (h *ImportHandler) importSingleImage(localImage, destImage string, auth authn.Authenticator) ImportResult {
	result := ImportResult{
		LocalImage:       localImage,
		DestinationImage: destImage,
	}

	// Créer une référence pour l'image locale
	localRef, err := name.ParseReference(localImage)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse local image reference: %w", err)
		return result
	}

	// Créer une référence pour l'image de destination
	destRef, err := name.ParseReference(destImage)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse destination image reference: %w", err)
		return result
	}

	// Options pour l'import
	opts := []remote.Option{
		remote.WithAuth(auth),
		remote.WithContext(h.ctx),
	}

	// Charger l'image depuis le système local
	descriptor, err := remote.Get(localRef, opts...)
	if err != nil {
		result.Error = fmt.Errorf("failed to load local image: %w", err)
		return result
	}

	// Obtenir l'image depuis le descriptor
	img, err := descriptor.Image()
	if err != nil {
		result.Error = fmt.Errorf("failed to get image from descriptor: %w", err)
		return result
	}

	// Pousser l'image vers le registry de destination
	if err := remote.Write(destRef, img, opts...); err != nil {
		result.Error = fmt.Errorf("failed to push image: %w", err)
		return result
	}

	// Log le succès si verbose
	if h.options.VerboseLevel > 0 {
		h.logger.Printf("Successfully imported image: %s -> %s", localImage, destImage)
	}

	return result
}
