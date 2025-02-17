package internal

import (
	"fmt"
	"path/filepath"
	"strings"

	brmsparser "github.com/Caezarr-OSS/brms-parser/brms"
)

// Config représente la configuration complète pour la migration d'images
type Config struct {
	Blocks []*Block
}

// Block représente un bloc de migration entre deux registries
type Block struct {
	SourceRegistry      Registry
	DestinationRegistry Registry
	ImageMappings       []ImageMapping
	Exclusions          []string
}

// Registry représente un registry d'images
type Registry struct {
	Host string // Le nom d'hôte du registry (e.g., "registry.example.com")
}

// ImageMapping représente le mapping entre une image source et une image destination
type ImageMapping struct {
	Source      string
	Destination string
}

// ParseConfig parse un fichier BRMS et retourne la configuration
func ParseConfig(configPath string) (*Config, error) {
	// Convertir en chemin absolu
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Créer un nouveau parser BRMS
	parser := brmsparser.NewParser(absPath, brmsparser.LogLevelInfo)

	// Parser le fichier BRMS
	parsed, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse BRMS file: %w", err)
	}

	// Convertir en configuration
	config := &Config{
		Blocks: make([]*Block, 0),
	}

	// Traiter chaque bloc
	for sourceReg, destReg := range parsed.Blocks {
		// Parser les registries source et destination
		sourceRegistry, err := parseRegistryURL(sourceReg)
		if err != nil {
			return nil, fmt.Errorf("invalid source registry: %w", err)
		}

		destRegistry, err := parseRegistryURL(destReg)
		if err != nil {
			return nil, fmt.Errorf("invalid destination registry: %w", err)
		}

		// Créer les mappings d'images
		mappings := make([]ImageMapping, 0)
		for _, mapping := range parsed.Entities {
			mappings = append(mappings, ImageMapping{
				Source:      mapping.Source,
				Destination: mapping.Destination,
			})
		}

		// Créer les exclusions
		exclusions := make([]string, 0)
		for _, exclusion := range parsed.IgnoredItems {
			exclusions = append(exclusions, exclusion.Source)
		}

		// Ajouter le bloc à la configuration
		config.Blocks = append(config.Blocks, &Block{
			SourceRegistry:      sourceRegistry,
			DestinationRegistry: destRegistry,
			ImageMappings:       mappings,
			Exclusions:          exclusions,
		})
	}

	return config, nil
}

// parseRegistryURL parse une URL de registry et retourne une structure Registry
func parseRegistryURL(url string) (Registry, error) {
	// Nettoyer l'URL
	url = strings.TrimSpace(url)
	if url == "" {
		return Registry{}, fmt.Errorf("registry URL cannot be empty")
	}

	// Retirer le protocole s'il existe
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	// Retirer les slashes au début et à la fin
	url = strings.Trim(url, "/")

	return Registry{
		Host: url,
	}, nil
}
