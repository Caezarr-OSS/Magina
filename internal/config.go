package internal

import (
	"fmt"
	"path/filepath"
	"strings"

	brmsparser "github.com/Caezarr-OSS/brms-parser/brms"
)

// Config represents the complete configuration for image migration
type Config struct {
	Blocks []*Block
}

// Block represents a migration block between two registries
type Block struct {
	SourceRegistry      Registry
	DestinationRegistry Registry
	ImageMappings       []ImageMapping
	Exclusions          []string
}

// Registry represents an image registry
type Registry struct {
	Host string // The registry hostname (e.g., "registry.example.com")
}

// ImageMapping represents the mapping between a source and destination image
type ImageMapping struct {
	Source      string
	Destination string
}

// ParseConfig parses a BRMS file and returns the configuration
func ParseConfig(configPath string) (*Config, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create a new BRMS parser
	parser := brmsparser.NewParser(absPath, brmsparser.LogLevelInfo)

	// Parse the BRMS file
	parsed, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse BRMS file: %w", err)
	}

	// Convert to configuration
	config := &Config{
		Blocks: make([]*Block, 0),
	}

	// Process each block
	for sourceReg, destReg := range parsed.Blocks {
		// Parse source and destination registries
		sourceRegistry, err := parseRegistryURL(sourceReg)
		if err != nil {
			return nil, fmt.Errorf("invalid source registry: %w", err)
		}

		destRegistry, err := parseRegistryURL(destReg)
		if err != nil {
			return nil, fmt.Errorf("invalid destination registry: %w", err)
		}

		// Create image mappings
		mappings := make([]ImageMapping, 0)
		for _, mapping := range parsed.Entities {
			mappings = append(mappings, ImageMapping{
				Source:      mapping.Source,
				Destination: mapping.Destination,
			})
		}

		// Create exclusions
		exclusions := make([]string, 0)
		for _, exclusion := range parsed.IgnoredItems {
			exclusions = append(exclusions, exclusion.Source)
		}

		// Add block to configuration
		config.Blocks = append(config.Blocks, &Block{
			SourceRegistry:      sourceRegistry,
			DestinationRegistry: destRegistry,
			ImageMappings:       mappings,
			Exclusions:          exclusions,
		})
	}

	return config, nil
}

// parseRegistryURL parses a registry URL and returns a Registry structure
func parseRegistryURL(url string) (Registry, error) {
	// Clean URL
	url = strings.TrimSpace(url)
	if url == "" {
		return Registry{}, fmt.Errorf("registry URL cannot be empty")
	}

	// Remove protocol if present
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	// Remove leading and trailing slashes
	url = strings.Trim(url, "/")

	return Registry{
		Host: url,
	}, nil
}
