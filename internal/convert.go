package internal

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	LocalImage       string
	DestinationImage string
	Error            error
}

// ConvertHandler gère la conversion d'images
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

// ConvertImages convertit les images depuis le stockage local vers le format de destination
func (h *ConvertHandler) ConvertImages(block *Block) <-chan ConvertResult {
	results := make(chan ConvertResult)

	go func() {
		defer close(results)

		// Valider que le bloc est valide pour la conversion
		if err := h.validateConvertBlock(block); err != nil {
			results <- ConvertResult{Error: err}
			return
		}

		// Traiter chaque mapping d'image
		for _, mapping := range block.ImageMappings {
			// Vérifier si l'image est exclue
			if h.isExcluded(mapping.Source, block.Exclusions) {
				continue
			}

			// Convertir l'image
			result := h.convertSingleImage(mapping.Source, mapping.Source, mapping.Destination)
			results <- result
		}
	}()

	return results
}

// validateConvertBlock vérifie que le bloc est valide pour la conversion
func (h *ConvertHandler) validateConvertBlock(block *Block) error {
	if block == nil {
		return fmt.Errorf("le bloc ne peut pas être nil")
	}

	if block.DestinationRegistry.Host == "" {
		return fmt.Errorf("l'hôte du registre de destination ne peut pas être vide")
	}

	if len(block.ImageMappings) == 0 {
		return fmt.Errorf("aucun mapping d'image trouvé")
	}

	return nil
}

// isExcluded vérifie si une image est dans la liste des exclusions
func (h *ConvertHandler) isExcluded(image string, exclusions []string) bool {
	for _, exclusion := range exclusions {
		if strings.HasPrefix(exclusion, "!") {
			pattern := exclusion[1:]
			if strings.Contains(image, pattern) {
				return true
			}
		}
	}
	return false
}

// convertSingleImage convertit une seule image
func (h *ConvertHandler) convertSingleImage(sourceImage, localImage, destinationImage string) ConvertResult {
	result := ConvertResult{
		SourceImage:      sourceImage,
		LocalImage:       localImage,
		DestinationImage: destinationImage,
	}

	// Créer une référence pour l'image locale
	localRef, err := name.ParseReference(localImage)
	if err != nil {
		result.Error = fmt.Errorf("échec de l'analyse de la référence de l'image locale : %w", err)
		return result
	}

	// Créer une référence pour l'image de destination
	destRef, err := name.ParseReference(destinationImage)
	if err != nil {
		result.Error = fmt.Errorf("échec de l'analyse de la référence de l'image de destination : %w", err)
		return result
	}

	// Options pour la conversion
	opts := []remote.Option{
		remote.WithContext(h.ctx),
	}

	// Charger l'image depuis le stockage local
	img, err := remote.Image(localRef, opts...)
	if err != nil {
		result.Error = fmt.Errorf("échec du chargement de l'image locale : %w", err)
		return result
	}

	// Enregistrer l'image avec la nouvelle référence
	if err := remote.Write(destRef, img, opts...); err != nil {
		result.Error = fmt.Errorf("échec de l'écriture de l'image : %w", err)
		return result
	}

	// Journaliser la réussite si verbose
	if h.options.VerboseLevel > 0 {
		h.logger.Printf("Conversion d'image réussie : %s -> %s", localImage, destinationImage)
	}

	return result
}
