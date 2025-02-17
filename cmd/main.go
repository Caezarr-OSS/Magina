package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/caezarr-oss/magina/internal"
	"github.com/spf13/cobra"
)

var (
	version       = "dev"
	cfgFile       string
	verboseLevel  int
	cleanOnError  bool
	resumeOnError bool
	rootCmd       *cobra.Command
	exportCmd     *cobra.Command
	importCmd     *cobra.Command
	convertCmd    *cobra.Command
	transferCmd   *cobra.Command
	validateCmd   *cobra.Command
	session       *internal.Session
)

func init() {
	// Initialiser la session
	session = internal.NewSession()

	rootCmd = &cobra.Command{
		Use:     "magina",
		Short:   "Manage OCI images between registries",
		Long:    `Magina is a tool for managing OCI images between registries using BRMS configuration.`,
		Version: version,
	}

	// Commande export
	exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Export images from source registry to local host",
		Long: `Export images from the source registry specified in the BRMS configuration to the local host.
The configuration file must contain exactly one block with the source registry information.
Format: [protocol://export-host|]
Example: magina export -c config.brms`,
		RunE: handleExport,
	}

	// Commande convert
	convertCmd = &cobra.Command{
		Use:   "convert",
		Short: "Convert local images to new tags",
		Long: `Convert (retag) local images according to the BRMS configuration.
Requires images to be present locally from a previous export operation.
Format: [protocol://source-host|protocol://dest-host]
Example: magina convert -c config.brms`,
		RunE: handleConvert,
	}

	// Commande import
	importCmd = &cobra.Command{
		Use:   "import",
		Short: "Import local images to destination registry",
		Long: `Import local images to the destination registry specified in the BRMS configuration.
The configuration file must contain exactly one block with the destination registry information.
Requires images to be present locally with correct tags from previous convert operation.
Format: [|protocol://import-host]
Example: magina import -c config.brms`,
		RunE: handleImport,
	}

	// Commande transfer
	transferCmd = &cobra.Command{
		Use:   "transfer",
		Short: "Complete transfer workflow (export + convert + import)",
		Long: `Execute the complete transfer workflow:
1. Export images from source registry to local host
2. Convert (retag) images locally
3. Import images to destination registry
Format: [protocol://source-host|protocol://dest-host]
Example: magina transfer -c config.brms`,
		RunE: handleTransfer,
	}

	// Commande validate
	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate BRMS configuration file",
		Long: `Validate a BRMS configuration file without performing any operations.
Checks:
- Syntax validation
- Protocol specification
- Single block requirement
- Registry accessibility
Example: magina validate -c config.brms`,
		RunE: handleValidate,
	}

	// Flags globaux
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "BRMS configuration file (required)")
	rootCmd.MarkPersistentFlagRequired("config")
	rootCmd.PersistentFlags().IntVarP(&verboseLevel, "verbose", "v", 0, "Verbose level (0-3)")

	// Flags pour les commandes de transfert
	for _, cmd := range []*cobra.Command{exportCmd, importCmd, convertCmd, transferCmd} {
		cmd.Flags().BoolVar(&cleanOnError, "clean-on-error", false, "Clean downloaded/converted images on error")
		cmd.Flags().BoolVar(&resumeOnError, "resume", false, "Try to resume from last successful operation")
	}

	// Ajouter les sous-commandes
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(transferCmd)
	rootCmd.AddCommand(validateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Les handlers seront implémentés dans des fichiers séparés
func handleExport(cmd *cobra.Command, args []string) error {
	config, err := internal.ParseConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if len(config.Blocks) != 1 {
		return fmt.Errorf("export requires exactly one block in configuration, found %d", len(config.Blocks))
	}

	block := config.Blocks[0]

	// Récupérer les credentials pour le registry source
	creds, err := session.GetCredentials(block.SourceRegistry.Host)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Créer les options d'export
	options := internal.ExportOptions{
		CleanOnError: cleanOnError,
		VerboseLevel: verboseLevel,
		Credentials:  creds,
	}

	// Créer le gestionnaire d'export
	handler := internal.NewExportHandler(cmd.Context(), options)

	// Lancer l'export
	results := handler.ExportImages(block)

	// Compteurs pour le suivi
	var totalImages, successCount, failureCount int

	// Traiter les résultats
	for result := range results {
		totalImages++
		if result.Error != nil {
			failureCount++
			fmt.Printf("❌ FAILED  %s\n", result.SourceImage)
			if verboseLevel > 0 {
				fmt.Printf("   Error: %v\n", result.Error)
			}
		} else {
			successCount++
			if verboseLevel > 0 {
				fmt.Printf("✅ SUCCESS %s -> %s\n", result.SourceImage, result.LocalImage)
			}
		}
	}

	// Afficher le résumé
	fmt.Printf("\nExport Summary:\n")
	fmt.Printf("Total Images:  %d\n", totalImages)
	fmt.Printf("Successful:    %d\n", successCount)
	fmt.Printf("Failed:        %d\n", failureCount)

	if failureCount > 0 {
		return fmt.Errorf("%d images failed to export", failureCount)
	}

	return nil
}

func handleConvert(cmd *cobra.Command, args []string) error {
	config, err := internal.ParseConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if len(config.Blocks) != 1 {
		return fmt.Errorf("convert requires exactly one block in configuration, found %d", len(config.Blocks))
	}

	block := config.Blocks[0]

	// Créer les options de conversion
	options := internal.ConvertOptions{
		CleanOnError: cleanOnError,
		VerboseLevel: verboseLevel,
	}

	// Créer le gestionnaire de conversion
	handler := internal.NewConvertHandler(cmd.Context(), options)

	// Lancer la conversion
	results := handler.ConvertImages(block)

	// Compteurs pour le suivi
	var totalImages, successCount, failureCount int

	// Traiter les résultats
	for result := range results {
		totalImages++
		if result.Error != nil {
			failureCount++
			fmt.Printf("❌ FAILED  %s -> %s\n", result.SourceImage, result.DestinationImage)
			if verboseLevel > 0 {
				fmt.Printf("   Error: %v\n", result.Error)
			}
		} else {
			successCount++
			if verboseLevel > 0 {
				fmt.Printf("✅ SUCCESS %s -> %s\n", result.SourceImage, result.DestinationImage)
			}
		}
	}

	// Afficher le résumé
	fmt.Printf("\nConversion Summary:\n")
	fmt.Printf("Total Images:  %d\n", totalImages)
	fmt.Printf("Successful:    %d\n", successCount)
	fmt.Printf("Failed:        %d\n", failureCount)

	if failureCount > 0 {
		return fmt.Errorf("%d images failed to convert", failureCount)
	}

	return nil
}

func handleImport(cmd *cobra.Command, args []string) error {
	config, err := internal.ParseConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if len(config.Blocks) != 1 {
		return fmt.Errorf("import requires exactly one block in configuration, found %d", len(config.Blocks))
	}

	block := config.Blocks[0]

	// Récupérer les credentials pour le registry de destination
	creds, err := session.GetCredentials(block.DestinationRegistry.Host)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Créer les options d'import
	options := internal.ImportOptions{
		CleanOnError: cleanOnError,
		VerboseLevel: verboseLevel,
		Credentials:  creds,
	}

	// Créer le gestionnaire d'import
	handler := internal.NewImportHandler(cmd.Context(), options)

	// Lancer l'import
	results := handler.ImportImages(block)

	// Compteurs pour le suivi
	var totalImages, successCount, failureCount int

	// Traiter les résultats
	for result := range results {
		totalImages++
		if result.Error != nil {
			failureCount++
			fmt.Printf("❌ FAILED  %s\n", result.DestinationImage)
			if verboseLevel > 0 {
				fmt.Printf("   Error: %v\n", result.Error)
			}
		} else {
			successCount++
			if verboseLevel > 0 {
				fmt.Printf("✅ SUCCESS %s\n", result.DestinationImage)
			}
		}
	}

	// Afficher le résumé
	fmt.Printf("\nImport Summary:\n")
	fmt.Printf("Total Images:  %d\n", totalImages)
	fmt.Printf("Successful:    %d\n", successCount)
	fmt.Printf("Failed:        %d\n", failureCount)

	if failureCount > 0 {
		return fmt.Errorf("%d images failed to import", failureCount)
	}

	return nil
}

func handleTransfer(cmd *cobra.Command, args []string) error {
	config, err := internal.ParseConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if len(config.Blocks) != 1 {
		return fmt.Errorf("transfer requires exactly one block in configuration, found %d", len(config.Blocks))
	}

	block := config.Blocks[0]

	// Créer les options de transfert
	options := internal.TransferOptions{
		CleanOnError:  cleanOnError,
		VerboseLevel:  verboseLevel,
		ResumeOnError: resumeOnError,
	}

	// Créer le gestionnaire de transfert
	handler := internal.NewTransferHandler(cmd.Context(), options, session)

	// Lancer le transfert
	results := handler.TransferImages(block)

	// Compteurs pour le suivi
	counters := make(map[internal.TransferPhase]struct {
		total    int
		success  int
		failures int
	})

	// Traiter les résultats
	for result := range results {
		phase := result.Phase
		stats := counters[phase]
		stats.total++

		if result.Error != nil {
			stats.failures++
			fmt.Printf("❌ %s FAILED  ", phase)
			if result.SourceImage != "" {
				fmt.Printf("%s", result.SourceImage)
			}
			if result.DestinationImage != "" {
				fmt.Printf(" -> %s", result.DestinationImage)
			}
			fmt.Println()
			if verboseLevel > 0 {
				fmt.Printf("   Error: %v\n", result.Error)
			}
		} else {
			stats.success++
			if verboseLevel > 0 {
				fmt.Printf("✅ %s SUCCESS  ", phase)
				if result.SourceImage != "" {
					fmt.Printf("%s", result.SourceImage)
				}
				if result.DestinationImage != "" {
					fmt.Printf(" -> %s", result.DestinationImage)
				}
				fmt.Println()
			}
		}

		counters[phase] = stats
	}

	// Afficher le résumé
	fmt.Printf("\nTransfer Summary:\n")
	var totalFailures int

	for _, phase := range []internal.TransferPhase{
		internal.PhaseExport,
		internal.PhaseConvert,
		internal.PhaseImport,
	} {
		stats := counters[phase]
		if stats.total > 0 {
			fmt.Printf("\n%s Phase:\n", phase)
			fmt.Printf("  Total:      %d\n", stats.total)
			fmt.Printf("  Successful: %d\n", stats.success)
			fmt.Printf("  Failed:     %d\n", stats.failures)
			totalFailures += stats.failures
		}
	}

	if totalFailures > 0 {
		return fmt.Errorf("transfer completed with %d total failures", totalFailures)
	}

	return nil
}

func handleValidate(cmd *cobra.Command, args []string) error {
	config, err := internal.ParseConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(config.Blocks) != 1 {
		return fmt.Errorf("configuration must contain exactly one block, found %d", len(config.Blocks))
	}

	block := config.Blocks[0]
	
	// Vérifier que le protocol est spécifié
	if !strings.HasPrefix(block.SourceRegistry.Host, "http://") && 
	   !strings.HasPrefix(block.SourceRegistry.Host, "https://") {
		return fmt.Errorf("source registry URL must specify protocol (http:// or https://)")
	}

	if block.DestinationRegistry.Host != "" {
		if !strings.HasPrefix(block.DestinationRegistry.Host, "http://") && 
		   !strings.HasPrefix(block.DestinationRegistry.Host, "https://") {
			return fmt.Errorf("destination registry URL must specify protocol (http:// or https://)")
		}
	}

	fmt.Printf("✅ Configuration is valid!\n\n")
	fmt.Printf("Source Registry:      %s\n", block.SourceRegistry.Host)
	if block.DestinationRegistry.Host != "" {
		fmt.Printf("Destination Registry: %s\n", block.DestinationRegistry.Host)
	}
	fmt.Printf("Number of Images:     %d\n", len(block.ImageMappings))
	if len(block.Exclusions) > 0 {
		fmt.Printf("Exclusions:          %d\n", len(block.Exclusions))
	}

	return nil
}
