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

// ExportOptions contient les options pour l'opération d'export
type ExportOptions struct {
	CleanOnError bool
	VerboseLevel int
	Credentials  *Credentials
}

// ExportResult représente le résultat d'un export d'image
type ExportResult struct {
	SourceImage  string
	LocalImage   string
	Error        error
}

// ExportHandler gère l'export des images
type ExportHandler struct {
	ctx     context.Context
	options ExportOptions
	logger  *log.Logger
}

// NewExportHandler crée un nouveau gestionnaire d'export
func NewExportHandler(ctx context.Context, options ExportOptions) *ExportHandler {
	return &ExportHandler{
		ctx:     ctx,
		options: options,
		logger:  log.New(log.Writer(), "[EXPORT] ", log.LstdFlags),
	}
}

// ExportImages exporte les images depuis le registry source
func (h *ExportHandler) ExportImages(block *Block) <-chan ExportResult {
	results := make(chan ExportResult)

	go func() {
		defer close(results)

		// Vérifier que le bloc est valide pour l'export
		if err := h.validateExportBlock(block); err != nil {
			results <- ExportResult{Error: err}
			return
		}

		// Configurer l'authentification
		auth := h.getAuthConfig(block.SourceRegistry.Host)

		// Traiter chaque mapping d'image
		for _, mapping := range block.ImageMappings {
			// Vérifier si l'image est exclue
			if h.isExcluded(mapping.Source, block.Exclusions) {
				continue
			}

			// Exporter l'image
			result := h.exportSingleImage(mapping.Source, mapping.Destination, auth)
			results <- result
		}
	}()

	return results
}

// validateExportBlock vérifie que le bloc est valide pour l'export
func (h *ExportHandler) validateExportBlock(block *Block) error {
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}

	if block.SourceRegistry.Host == "" {
		return fmt.Errorf("source registry host cannot be empty")
	}

	if len(block.ImageMappings) == 0 {
		return fmt.Errorf("no image mappings found")
	}

	return nil
}

// getAuthConfig configure l'authentification pour le registry
func (h *ExportHandler) getAuthConfig(registryURL string) authn.Authenticator {
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
func (h *ExportHandler) isExcluded(image string, exclusions []string) bool {
	for _, exclusion := range exclusions {
		if strings.Contains(image, exclusion) {
			return true
		}
	}
	return false
}

// exportSingleImage exporte une seule image
func (h *ExportHandler) exportSingleImage(sourceImage, localImage string, auth authn.Authenticator) ExportResult {
	result := ExportResult{
		SourceImage: sourceImage,
		LocalImage:  localImage,
	}

	// Créer une référence pour l'image source
	sourceRef, err := name.ParseReference(sourceImage)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse source image reference: %w", err)
		return result
	}

	// Créer une référence pour l'image locale
	localRef, err := name.ParseReference(localImage)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse local image reference: %w", err)
		return result
	}

	// Options pour l'export
	opts := []remote.Option{
		remote.WithAuth(auth),
		remote.WithContext(h.ctx),
	}

	// Charger l'image depuis le registry source
	descriptor, err := remote.Get(sourceRef, opts...)
	if err != nil {
		result.Error = fmt.Errorf("failed to load source image: %w", err)
		return result
	}

	// Obtenir l'image depuis le descriptor
	img, err := descriptor.Image()
	if err != nil {
		result.Error = fmt.Errorf("failed to get image from descriptor: %w", err)
		return result
	}

	// Sauvegarder l'image localement
	if err := remote.Write(localRef, img, opts...); err != nil {
		result.Error = fmt.Errorf("failed to save image locally: %w", err)
		return result
	}

	// Log le succès si verbose
	if h.options.VerboseLevel > 0 {
		h.logger.Printf("Successfully exported image: %s -> %s", sourceImage, localImage)
	}

	return result
}
