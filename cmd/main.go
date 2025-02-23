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
		Short:   "Gérer les images OCI entre les registres",
		Long:    `Magina est un outil pour gérer les images OCI entre les registres en utilisant la configuration BRMS.`,
		Version: version,
	}

	// Commande d'exportation
	exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Exporter les images du registre source vers l'hôte local",
		Long: `Exporter les images du registre source spécifié dans la configuration BRMS vers l'hôte local.
La configuration doit contenir exactement un bloc avec les informations du registre source.
Format : [protocole://export-host|]
Exemple : magina export -c config.brms`,
		RunE: handleExport,
	}

	// Commande de conversion
	convertCmd = &cobra.Command{
		Use:   "convert",
		Short: "Convertir les images locales en nouveaux tags",
		Long: `Convertir (retaguer) les images locales selon la configuration BRMS.
Nécessite la présence des images locales à partir d'une opération d'exportation précédente.
Format : [protocole://source-host|protocole://dest-host]
Exemple : magina convert -c config.brms`,
		RunE: handleConvert,
	}

	// Commande d'importation
	importCmd = &cobra.Command{
		Use:   "import",
		Short: "Importer les images locales vers le registre de destination",
		Long: `Importer les images locales vers le registre de destination spécifié dans la configuration BRMS.
La configuration doit contenir exactement un bloc avec les informations du registre de destination.
Nécessite la présence des images locales avec les tags corrects à partir d'une opération de conversion précédente.
Format : [|protocole://import-host]
Exemple : magina import -c config.brms`,
		RunE: handleImport,
	}

	// Commande de transfert
	transferCmd = &cobra.Command{
		Use:   "transfer",
		Short: "Effectuer le workflow de transfert complet (export + conversion + importation)",
		Long: `Exécuter le workflow de transfert complet :
1. Exporter les images du registre source vers l'hôte local
2. Convertir (retaguer) les images locales
3. Importer les images vers le registre de destination
Format : [protocole://source-host|protocole://dest-host]
Exemple : magina transfer -c config.brms`,
		RunE: handleTransfer,
	}

	// Commande de validation
	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Valider le fichier de configuration BRMS",
		Long: `Valider un fichier de configuration BRMS sans effectuer d'opérations.
Vérifications :
- Validation de la syntaxe
- Spécification du protocole
- Exigence d'un seul bloc
- Accessibilité du registre
Exemple : magina validate -c config.brms`,
		RunE: handleValidate,
	}

	// Flags globaux
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Fichier de configuration BRMS (obligatoire)")
	rootCmd.MarkPersistentFlagRequired("config")
	rootCmd.PersistentFlags().IntVarP(&verboseLevel, "verbose", "v", 0, "Niveau de verbosité (0-3)")

	// Flags pour les commandes de transfert
	for _, cmd := range []*cobra.Command{exportCmd, importCmd, convertCmd, transferCmd} {
		cmd.Flags().BoolVar(&cleanOnError, "clean-on-error", false, "Nettoyer les images téléchargées/converties en cas d'erreur")
		cmd.Flags().BoolVar(&resumeOnError, "resume", false, "Essayer de reprendre à partir de la dernière opération réussie")
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

// Les gestionnaires seront implémentés dans des fichiers séparés
func handleExport(cmd *cobra.Command, args []string) error {
	config, err := internal.ParseConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("échec de l'analyse de la configuration : %w", err)
	}

	if len(config.Blocks) != 1 {
		return fmt.Errorf("l'export nécessite exactement un bloc dans la configuration, trouvé %d", len(config.Blocks))
	}

	block := config.Blocks[0]

	// Obtenir les informations d'identification pour le registre source
	creds, err := session.GetCredentials(block.SourceRegistry.Host)
	if err != nil {
		return fmt.Errorf("échec de l'obtention des informations d'identification : %w", err)
	}

	// Créer les options d'exportation
	options := internal.ExportOptions{
		CleanOnError: cleanOnError,
		VerboseLevel: verboseLevel,
		Credentials:  creds,
	}

	// Créer le gestionnaire d'exportation
	handler := internal.NewExportHandler(cmd.Context(), options)

	// Démarrer l'exportation
	results := handler.ExportImages(block)

	// Compteurs pour le suivi
	var totalImages, successCount, failureCount int

	// Traiter les résultats
	for result := range results {
		totalImages++
		if result.Error != nil {
			failureCount++
			fmt.Printf("❌ ÉCHEC  %s\n", result.SourceImage)
			if verboseLevel > 0 {
				fmt.Printf("   Erreur : %v\n", result.Error)
			}
		} else {
			successCount++
			if verboseLevel > 0 {
				fmt.Printf("✅ SUCCÈS %s -> %s\n", result.SourceImage, result.LocalImage)
			}
		}
	}

	// Afficher le résumé
	fmt.Printf("\nRésumé de l'exportation :\n")
	fmt.Printf("Total des images :  %d\n", totalImages)
	fmt.Printf("Réussites :    %d\n", successCount)
	fmt.Printf("Échecs :        %d\n", failureCount)

	if failureCount > 0 {
		return fmt.Errorf("%d images n'ont pas pu être exportées", failureCount)
	}

	return nil
}

func handleConvert(cmd *cobra.Command, args []string) error {
	config, err := internal.ParseConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("échec de l'analyse de la configuration : %w", err)
	}

	if len(config.Blocks) != 1 {
		return fmt.Errorf("la conversion nécessite exactement un bloc dans la configuration, trouvé %d", len(config.Blocks))
	}

	block := config.Blocks[0]

	// Créer les options de conversion
	options := internal.ConvertOptions{
		CleanOnError: cleanOnError,
		VerboseLevel: verboseLevel,
	}

	// Créer le gestionnaire de conversion
	handler := internal.NewConvertHandler(cmd.Context(), options)

	// Démarrer la conversion
	results := handler.ConvertImages(block)

	// Compteurs pour le suivi
	var totalImages, successCount, failureCount int

	// Traiter les résultats
	for result := range results {
		totalImages++
		if result.Error != nil {
			failureCount++
			fmt.Printf("❌ ÉCHEC  %s -> %s : %v\n", result.SourceImage, result.DestinationImage, result.Error)
			return fmt.Errorf("échec de la conversion de l'image : %w", result.Error)
		}

		successCount++
		fmt.Printf("✅ SUCCÈS %s -> %s\n", result.SourceImage, result.DestinationImage)
	}

	// Afficher le résumé
	fmt.Printf("\nRésumé de la conversion :\n")
	fmt.Printf("Total des images :  %d\n", totalImages)
	fmt.Printf("Réussites :    %d\n", successCount)
	fmt.Printf("Échecs :        %d\n", failureCount)

	if failureCount > 0 {
		return fmt.Errorf("%d images n'ont pas pu être converties", failureCount)
	}

	return nil
}

func handleImport(cmd *cobra.Command, args []string) error {
	config, err := internal.ParseConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("échec de l'analyse de la configuration : %w", err)
	}

	if len(config.Blocks) != 1 {
		return fmt.Errorf("l'importation nécessite exactement un bloc dans la configuration, trouvé %d", len(config.Blocks))
	}

	block := config.Blocks[0]

	// Obtenir les informations d'identification pour le registre de destination
	creds, err := session.GetCredentials(block.DestinationRegistry.Host)
	if err != nil {
		return fmt.Errorf("échec de l'obtention des informations d'identification : %w", err)
	}

	// Créer les options d'importation
	options := internal.ImportOptions{
		CleanOnError: cleanOnError,
		VerboseLevel: verboseLevel,
		Credentials:  creds,
	}

	// Créer le gestionnaire d'importation
	handler := internal.NewImportHandler(cmd.Context(), options)

	// Démarrer l'importation
	results := handler.ImportImages(block)

	// Compteurs pour le suivi
	var totalImages, successCount, failureCount int

	// Traiter les résultats
	for result := range results {
		totalImages++
		if result.Error != nil {
			failureCount++
			fmt.Printf("❌ ÉCHEC  %s\n", result.DestinationImage)
			if verboseLevel > 0 {
				fmt.Printf("   Erreur : %v\n", result.Error)
			}
		} else {
			successCount++
			if verboseLevel > 0 {
				fmt.Printf("✅ SUCCÈS %s\n", result.DestinationImage)
			}
		}
	}

	// Afficher le résumé
	fmt.Printf("\nRésumé de l'importation :\n")
	fmt.Printf("Total des images :  %d\n", totalImages)
	fmt.Printf("Réussites :    %d\n", successCount)
	fmt.Printf("Échecs :        %d\n", failureCount)

	if failureCount > 0 {
		return fmt.Errorf("%d images n'ont pas pu être importées", failureCount)
	}

	return nil
}

func handleTransfer(cmd *cobra.Command, args []string) error {
	config, err := internal.ParseConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("échec de l'analyse de la configuration : %w", err)
	}

	if len(config.Blocks) != 1 {
		return fmt.Errorf("le transfert nécessite exactement un bloc dans la configuration, trouvé %d", len(config.Blocks))
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

	// Démarrer le transfert
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
			fmt.Printf("❌ %s ÉCHEC  ", phase)
			if result.SourceImage != "" {
				fmt.Printf("%s", result.SourceImage)
			}
			if result.DestinationImage != "" {
				fmt.Printf(" -> %s", result.DestinationImage)
			}
			fmt.Println()
			if verboseLevel > 0 {
				fmt.Printf("   Erreur : %v\n", result.Error)
			}
		} else {
			stats.success++
			if verboseLevel > 0 {
				fmt.Printf("✅ %s SUCCÈS  ", phase)
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
	fmt.Printf("\nRésumé du transfert :\n")
	var totalFailures int

	for _, phase := range []internal.TransferPhase{
		internal.PhaseExport,
		internal.PhaseConvert,
		internal.PhaseImport,
	} {
		stats := counters[phase]
		if stats.total > 0 {
			fmt.Printf("\nPhase %s :\n", phase)
			fmt.Printf("  Total :      %d\n", stats.total)
			fmt.Printf("  Réussites :    %d\n", stats.success)
			fmt.Printf("  Échecs :     %d\n", stats.failures)
			totalFailures += stats.failures
		}
	}

	if totalFailures > 0 {
		return fmt.Errorf("le transfert s'est terminé avec %d échecs au total", totalFailures)
	}

	return nil
}

func handleValidate(cmd *cobra.Command, args []string) error {
	config, err := internal.ParseConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("échec de la validation : %w", err)
	}

	if len(config.Blocks) != 1 {
		return fmt.Errorf("la configuration doit contenir exactement un bloc, trouvé %d", len(config.Blocks))
	}

	block := config.Blocks[0]

	// Vérifier que le protocole est spécifié
	if !strings.HasPrefix(block.SourceRegistry.Host, "http://") &&
		!strings.HasPrefix(block.SourceRegistry.Host, "https://") {
		return fmt.Errorf("l'URL du registre source doit spécifier le protocole (http:// ou https://)")
	}

	if block.DestinationRegistry.Host != "" {
		if !strings.HasPrefix(block.DestinationRegistry.Host, "http://") &&
			!strings.HasPrefix(block.DestinationRegistry.Host, "https://") {
			return fmt.Errorf("l'URL du registre de destination doit spécifier le protocole (http:// ou https://)")
		}
	}

	fmt.Printf("✅ La configuration est valide !\n\n")
	fmt.Printf("Registre source :      %s\n", block.SourceRegistry.Host)
	if block.DestinationRegistry.Host != "" {
		fmt.Printf("Registre de destination : %s\n", block.DestinationRegistry.Host)
	}
	fmt.Printf("Nombre d'images :     %d\n", len(block.ImageMappings))
	if len(block.Exclusions) > 0 {
		fmt.Printf("Exclusions :          %d\n", len(block.Exclusions))
	}

	return nil
}
