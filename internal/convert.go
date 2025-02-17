package internal

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// ConvertOptions contient les options pour l'opération de conversion
type ConvertOptions struct {
	CleanOnError bool
	VerboseLevel int
}

// ConvertResult représente le résultat d'une conversion d'image
type ConvertResult struct {
	SourceImage      string
	DestinationImage string
	Error           error
}

// ConvertHandler gère la conversion (retagging) des images
type ConvertHandler struct {
	ctx     context.Context
	options ConvertOptions
	logger  *log.Logger
}

// NewConvertHandler crée un nouveau gestionnaire de conversion
func NewConvertHandler(ctx context.Context, options ConvertOptions) *ConvertHandler {
	return &ConvertHandler{
		ctx:     ctx,
		options: options,
		logger:  log.New(log.Writer(), "[CONVERT] ", log.LstdFlags),
	}
}

// ConvertImages convertit les images selon la configuration
func (h *ConvertHandler) ConvertImages(block *Block) <-chan ConvertResult {
	results := make(chan ConvertResult)

	go func() {
		defer close(results)

		// Vérifier que le bloc est valide pour la conversion
		if err := h.validateConvertBlock(block); err != nil {
			results <- ConvertResult{Error: err}
			return
		}

		// Traiter chaque mapping d'image
		for _, mapping := range block.ImageMappings {
			// Vérifier si l'image est exclue
			if h.isExcluded(mapping.Source, block.Exclusions) {
				if h.options.VerboseLevel > 0 {
					h.logger.Printf("Skipping excluded image: %s", mapping.Source)
				}
				continue
			}

			result := h.convertSingleImage(mapping)
			results <- result

			if result.Error != nil && h.options.CleanOnError {
				h.cleanupImage(result.DestinationImage)
			}
		}
	}()

	return results
}

// validateConvertBlock vérifie que le bloc est valide pour la conversion
func (h *ConvertHandler) validateConvertBlock(block *Block) error {
	if block == nil {
		return fmt.Errorf("block is nil")
	}

	if len(block.ImageMappings) == 0 {
		return fmt.Errorf("no images to convert")
	}

	// Vérifier que chaque mapping a une destination
	for _, mapping := range block.ImageMappings {
		if mapping.Destination == "" {
			return fmt.Errorf("image %s has no destination tag", mapping.Source)
		}
	}

	return nil
}

// isExcluded vérifie si une image est dans la liste des exclusions
func (h *ConvertHandler) isExcluded(image string, exclusions []string) bool {
	for _, excl := range exclusions {
		if image == excl {
			return true
		}
	}
	return false
}

// convertSingleImage convertit une seule image
func (h *ConvertHandler) convertSingleImage(mapping ImageMapping) ConvertResult {
	if h.options.VerboseLevel > 0 {
		h.logger.Printf("Converting image: %s -> %s", mapping.Source, mapping.Destination)
	}

	// Construire la référence source
	sourceRef, err := name.ParseReference(mapping.Source)
	if err != nil {
		return ConvertResult{
			SourceImage: mapping.Source,
			Error:      fmt.Errorf("invalid source reference: %w", err),
		}
	}

	// Construire la référence destination
	destRef, err := name.ParseReference(mapping.Destination)
	if err != nil {
		return ConvertResult{
			SourceImage: mapping.Source,
			Error:      fmt.Errorf("invalid destination reference: %w", err),
		}
	}

	// Récupérer l'image source depuis le registre local
	img, err := remote.Image(sourceRef)
	if err != nil {
		return ConvertResult{
			SourceImage:      mapping.Source,
			DestinationImage: mapping.Destination,
			Error:           fmt.Errorf("failed to get source image: %w", err),
		}
	}

	// Tagger l'image avec la nouvelle référence
	if err := remote.Write(destRef, img); err != nil {
		return ConvertResult{
			SourceImage:      mapping.Source,
			DestinationImage: mapping.Destination,
			Error:           fmt.Errorf("failed to tag image: %w", err),
		}
	}

	if h.options.VerboseLevel > 1 {
		h.logger.Printf("Successfully converted %s to %s", mapping.Source, mapping.Destination)
	}

	return ConvertResult{
		SourceImage:      mapping.Source,
		DestinationImage: mapping.Destination,
	}
}

// cleanupImage nettoie une image en cas d'erreur
func (h *ConvertHandler) cleanupImage(image string) {
	if h.options.VerboseLevel > 0 {
		h.logger.Printf("Cleaning up image: %s", image)
	}
	// Supprimer le tag local
	if ref, err := name.ParseReference(image); err == nil {
		if err := remote.Delete(ref); err != nil && h.options.VerboseLevel > 0 {
			h.logger.Printf("Failed to cleanup image %s: %v", image, err)
		}
	}
}
