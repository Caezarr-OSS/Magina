package internal

import (
	"context"
	"fmt"
	"log"
)

// TransferPhase represents a phase in the transfer process
type TransferPhase string

const (
	PhaseExport  TransferPhase = "EXPORT"
	PhaseConvert TransferPhase = "CONVERT"
	PhaseImport  TransferPhase = "IMPORT"
)

// TransferOptions contains options for the transfer process
type TransferOptions struct {
	CleanOnError  bool
	VerboseLevel  int
	ResumeOnError bool
}

// TransferResult represents the result of a transfer operation
type TransferResult struct {
	Phase            TransferPhase
	SourceImage      string
	LocalImage       string
	DestinationImage string
	Error            error
}

// TransferHandler manages the complete transfer workflow
type TransferHandler struct {
	ctx     context.Context
	options TransferOptions
	logger  *log.Logger
	session *Session
}

// NewTransferHandler creates a new TransferHandler instance
func NewTransferHandler(ctx context.Context, options TransferOptions, session *Session) *TransferHandler {
	return &TransferHandler{
		ctx:     ctx,
		options: options,
		logger:  log.New(log.Writer(), "[TRANSFER] ", log.LstdFlags),
		session: session,
	}
}

// TransferImages executes the complete transfer workflow
func (h *TransferHandler) TransferImages(block *Block) <-chan TransferResult {
	results := make(chan TransferResult)

	go func() {
		defer close(results)

		// Validate that the block is valid for transfer
		if err := h.validateTransferBlock(block); err != nil {
			results <- TransferResult{Error: err}
			return
		}

		// Get credentials for source registry
		sourceCreds, err := h.session.GetCredentials(block.SourceRegistry.Host)
		if err != nil {
			results <- TransferResult{
				Phase: PhaseExport,
				Error: fmt.Errorf("failed to get source credentials: %w", err),
			}
			return
		}

		// Get credentials for destination registry
		destCreds, err := h.session.GetCredentials(block.DestinationRegistry.Host)
		if err != nil {
			results <- TransferResult{
				Phase: PhaseImport,
				Error: fmt.Errorf("failed to get destination credentials: %w", err),
			}
			return
		}

		// Export phase
		exportOpts := ExportOptions{
			CleanOnError: h.options.CleanOnError,
			VerboseLevel: h.options.VerboseLevel,
			Credentials:  sourceCreds,
		}
		exportHandler := NewExportHandler(h.ctx, exportOpts)
		exportResults := exportHandler.ExportImages(block)
		for result := range exportResults {
			results <- TransferResult{
				Phase:       PhaseExport,
				SourceImage: result.SourceImage,
				LocalImage:  result.LocalImage,
				Error:       result.Error,
			}
			if result.Error != nil && !h.options.ResumeOnError {
				return
			}
		}

		// Convert phase
		convertOpts := ConvertOptions{
			CleanOnError: h.options.CleanOnError,
			VerboseLevel: h.options.VerboseLevel,
		}
		convertHandler := NewConvertHandler(h.ctx, convertOpts)
		convertResults := convertHandler.ConvertImages(block)
		for result := range convertResults {
			results <- TransferResult{
				Phase:            PhaseConvert,
				SourceImage:      result.SourceImage,
				LocalImage:       result.LocalImage,
				DestinationImage: result.DestinationImage,
				Error:            result.Error,
			}
			if result.Error != nil && !h.options.ResumeOnError {
				return
			}
		}

		// Import phase
		importOpts := ImportOptions{
			CleanOnError: h.options.CleanOnError,
			VerboseLevel: h.options.VerboseLevel,
			Credentials:  destCreds,
		}
		importHandler := NewImportHandler(h.ctx, importOpts)
		importResults := importHandler.ImportImages(block)
		for result := range importResults {
			results <- TransferResult{
				Phase:            PhaseImport,
				LocalImage:       result.LocalImage,
				DestinationImage: result.DestinationImage,
				Error:            result.Error,
			}
			if result.Error != nil && !h.options.ResumeOnError {
				return
			}
		}
	}()

	return results
}

// validateTransferBlock validates that the block is valid for transfer
func (h *TransferHandler) validateTransferBlock(block *Block) error {
	if block == nil {
		return fmt.Errorf("the block cannot be nil")
	}

	if block.SourceRegistry.Host == "" {
		return fmt.Errorf("the source registry host cannot be empty")
	}

	if block.DestinationRegistry.Host == "" {
		return fmt.Errorf("the destination registry host cannot be empty")
	}

	if len(block.ImageMappings) == 0 {
		return fmt.Errorf("no image mappings found")
	}

	return nil
}
