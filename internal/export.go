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

// ExportOptions contains the options for the export operation
type ExportOptions struct {
	CleanOnError bool
	VerboseLevel int
	Credentials  *Credentials
}

// ExportResult represents the result of an image export
type ExportResult struct {
	SourceImage  string
	LocalImage   string
	Error        error
}

// ExportHandler handles image exports
type ExportHandler struct {
	ctx     context.Context
	options ExportOptions
	logger  *log.Logger
}

// NewExportHandler creates a new export handler
func NewExportHandler(ctx context.Context, options ExportOptions) *ExportHandler {
	return &ExportHandler{
		ctx:     ctx,
		options: options,
		logger:  log.New(log.Writer(), "[EXPORT] ", log.LstdFlags),
	}
}

// ExportImages exports images from the source registry
func (h *ExportHandler) ExportImages(block *Block) <-chan ExportResult {
	results := make(chan ExportResult)

	go func() {
		defer close(results)

		// Validate that the block is valid for export
		if err := h.validateExportBlock(block); err != nil {
			results <- ExportResult{Error: err}
			return
		}

		// Configure authentication
		auth := h.getAuthConfig(block.SourceRegistry.Host)

		// Process each image mapping
		for _, mapping := range block.ImageMappings {
			// Check if the image is excluded
			if h.isExcluded(mapping.Source, block.Exclusions) {
				continue
			}

			// Export the image
			result := h.exportSingleImage(mapping.Source, mapping.Destination, auth)
			results <- result
		}
	}()

	return results
}

// validateExportBlock verifies that the block is valid for export
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

// getAuthConfig configures authentication for the registry
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

// isExcluded checks if an image is in the exclusion list
func (h *ExportHandler) isExcluded(image string, exclusions []string) bool {
	for _, exclusion := range exclusions {
		if strings.Contains(image, exclusion) {
			return true
		}
	}
	return false
}

// exportSingleImage exports a single image
func (h *ExportHandler) exportSingleImage(sourceImage, localImage string, auth authn.Authenticator) ExportResult {
	result := ExportResult{
		SourceImage: sourceImage,
		LocalImage:  localImage,
	}

	// Create a reference for the source image
	sourceRef, err := name.ParseReference(sourceImage)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse source image reference: %w", err)
		return result
	}

	// Create a reference for the local image
	localRef, err := name.ParseReference(localImage)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse local image reference: %w", err)
		return result
	}

	// Options for export
	opts := []remote.Option{
		remote.WithAuth(auth),
		remote.WithContext(h.ctx),
	}

	// Load the image from the source registry
	descriptor, err := remote.Get(sourceRef, opts...)
	if err != nil {
		result.Error = fmt.Errorf("failed to load source image: %w", err)
		return result
	}

	// Get the image from the descriptor
	img, err := descriptor.Image()
	if err != nil {
		result.Error = fmt.Errorf("failed to get image from descriptor: %w", err)
		return result
	}

	// Save the image locally
	if err := remote.Write(localRef, img, opts...); err != nil {
		result.Error = fmt.Errorf("failed to save image locally: %w", err)
		return result
	}

	// Log success if verbose
	if h.options.VerboseLevel > 0 {
		h.logger.Printf("Successfully exported image: %s -> %s", sourceImage, localImage)
	}

	return result
}
