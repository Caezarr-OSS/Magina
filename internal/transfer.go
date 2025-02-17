package internal

import (
	"context"
	"fmt"
	"log"
)

// TransferPhase représente une phase du transfert
type TransferPhase string

const (
	PhaseExport  TransferPhase = "EXPORT"
	PhaseConvert TransferPhase = "CONVERT"
	PhaseImport  TransferPhase = "IMPORT"
)

// TransferOptions contient les options pour l'opération de transfert
type TransferOptions struct {
	CleanOnError  bool
	VerboseLevel  int
	ResumeOnError bool
	Credentials   *Credentials
}

// TransferResult représente le résultat d'un transfert d'image
type TransferResult struct {
	Phase            TransferPhase
	SourceImage      string
	DestinationImage string
	Error            error
}

// TransferHandler gère le transfert des images
type TransferHandler struct {
	ctx     context.Context
	options TransferOptions
	logger  *log.Logger
	session *Session
}

// NewTransferHandler crée un nouveau gestionnaire de transfert
func NewTransferHandler(ctx context.Context, options TransferOptions, session *Session) *TransferHandler {
	return &TransferHandler{
		ctx:     ctx,
		options: options,
		logger:  log.New(log.Writer(), "[TRANSFER] ", log.LstdFlags),
		session: session,
	}
}

// TransferImages transfère les images d'un registry à un autre
func (h *TransferHandler) TransferImages(block *Block) <-chan TransferResult {
	results := make(chan TransferResult)

	go func() {
		defer close(results)

		// Vérifier que le bloc est valide
		if err := h.validateTransferBlock(block); err != nil {
			results <- TransferResult{Error: err}
			return
		}

		// Récupérer les credentials pour le registry source
		sourceCreds, err := h.session.GetCredentials(block.SourceRegistry.Host)
		if err != nil {
			results <- TransferResult{Error: fmt.Errorf("failed to get source credentials: %w", err)}
			return
		}

		// Récupérer les credentials pour le registry de destination
		destCreds, err := h.session.GetCredentials(block.DestinationRegistry.Host)
		if err != nil {
			results <- TransferResult{Error: fmt.Errorf("failed to get destination credentials: %w", err)}
			return
		}

		// Phase 1: Export
		exportOpts := ExportOptions{
			CleanOnError: h.options.CleanOnError,
			VerboseLevel: h.options.VerboseLevel,
			Credentials:  sourceCreds,
		}
		exportHandler := NewExportHandler(h.ctx, exportOpts)
		exportResults := exportHandler.ExportImages(block)

		for result := range exportResults {
			if result.Error != nil {
				results <- TransferResult{
					Phase:       PhaseExport,
					SourceImage: result.SourceImage,
					Error:      result.Error,
				}
				if !h.options.ResumeOnError {
					return
				}
			} else {
				results <- TransferResult{
					Phase:            PhaseExport,
					SourceImage:      result.SourceImage,
					DestinationImage: result.LocalImage,
				}
			}
		}

		// Phase 3: Import
		importOpts := ImportOptions{
			CleanOnError: h.options.CleanOnError,
			VerboseLevel: h.options.VerboseLevel,
			Credentials:  destCreds,
		}
		importHandler := NewImportHandler(h.ctx, importOpts)
		importResults := importHandler.ImportImages(block)

		for result := range importResults {
			if result.Error != nil {
				results <- TransferResult{
					Phase:            PhaseImport,
					SourceImage:      result.LocalImage,
					DestinationImage: result.DestinationImage,
					Error:           result.Error,
				}
				if !h.options.ResumeOnError {
					return
				}
			} else {
				results <- TransferResult{
					Phase:            PhaseImport,
					SourceImage:      result.LocalImage,
					DestinationImage: result.DestinationImage,
				}
			}
		}
	}()

	return results
}

// validateTransferBlock vérifie que le bloc est valide pour le transfert
func (h *TransferHandler) validateTransferBlock(block *Block) error {
	if block == nil {
		return fmt.Errorf("block cannot be nil")
	}

	if block.SourceRegistry.Host == "" {
		return fmt.Errorf("source registry host cannot be empty")
	}

	if block.DestinationRegistry.Host == "" {
		return fmt.Errorf("destination registry host cannot be empty")
	}

	if len(block.ImageMappings) == 0 {
		return fmt.Errorf("no image mappings found")
	}

	return nil
}
