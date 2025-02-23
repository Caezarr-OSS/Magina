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

// ImportResult represents the result of an import operation
type ImportResult struct {
	LocalImage       string
	DestinationImage string
	Error            error
}

// ImportOptions contains options for the import process
type ImportOptions struct {
	CleanOnError bool
	VerboseLevel int
	Credentials  *Credentials
}

// ImportHandler manages the import of images to a destination registry
type ImportHandler struct {
	ctx     context.Context
	options ImportOptions
	logger  *log.Logger
}

// NewImportHandler creates a new ImportHandler instance
func NewImportHandler(ctx context.Context, options ImportOptions) *ImportHandler {
	return &ImportHandler{
		ctx:     ctx,
		options: options,
		logger:  log.New(log.Writer(), "[IMPORT] ", log.LstdFlags),
	}
}

// ImportImages imports images from local storage to destination registry
func (h *ImportHandler) ImportImages(block *Block) <-chan ImportResult {
	results := make(chan ImportResult)

	go func() {
		defer close(results)

		// Validate that the block is valid for import
		if err := h.validateImportBlock(block); err != nil {
			results <- ImportResult{Error: err}
			return
		}

		// Configure authentication
		auth := h.getAuthConfig(block.DestinationRegistry.Host)

		// Process each image mapping
		for _, mapping := range block.ImageMappings {
			// Skip excluded images
			if h.isExcluded(mapping.Destination, block.Exclusions) {
				continue
			}

			// Import image
			result := h.importSingleImage(mapping.Source, mapping.Destination, auth)
			results <- result
		}
	}()

	return results
}

// validateImportBlock validates that the block is valid for import
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

// getAuthConfig configures authentication for the registry
func (h *ImportHandler) getAuthConfig(registryURL string) authn.Authenticator {
	if h.options.Credentials == nil {
		return authn.Anonymous
	}

	return authn.FromConfig(authn.AuthConfig{
		Username: h.options.Credentials.Username,
		Password: h.options.Credentials.Password,
	})
}

// isExcluded checks if an image is in the exclusion list
func (h *ImportHandler) isExcluded(image string, exclusions []string) bool {
	for _, exclusion := range exclusions {
		if strings.Contains(image, exclusion) {
			return true
		}
	}
	return false
}

// importSingleImage imports a single image
func (h *ImportHandler) importSingleImage(localImage, destImage string, auth authn.Authenticator) ImportResult {
	result := ImportResult{
		LocalImage:       localImage,
		DestinationImage: destImage,
	}

	// Create reference for local image
	localRef, err := name.ParseReference(localImage)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse local image reference: %w", err)
		return result
	}

	// Create reference for destination image
	destRef, err := name.ParseReference(destImage)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse destination image reference: %w", err)
		return result
	}

	// Options for import
	opts := []remote.Option{
		remote.WithAuth(auth),
		remote.WithContext(h.ctx),
	}

	// Load image from local storage
	descriptor, err := remote.Get(localRef, opts...)
	if err != nil {
		result.Error = fmt.Errorf("failed to load local image: %w", err)
		return result
	}

	// Get image from descriptor
	img, err := descriptor.Image()
	if err != nil {
		result.Error = fmt.Errorf("failed to get image from descriptor: %w", err)
		return result
	}

	// Push image to destination registry
	if err := remote.Write(destRef, img, opts...); err != nil {
		result.Error = fmt.Errorf("failed to push image: %w", err)
		return result
	}

	// Log success if verbose
	if h.options.VerboseLevel > 0 {
		h.logger.Printf("Image imported successfully: %s -> %s", localImage, destImage)
	}

	return result
}
