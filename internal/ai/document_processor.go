package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/rs/zerolog/log"
)

// DocumentProcessor handles document chunking and embedding
type DocumentProcessor struct {
	storage          *KnowledgeBaseStorage
	embeddingService *EmbeddingService
}

// NewDocumentProcessor creates a new document processor
func NewDocumentProcessor(storage *KnowledgeBaseStorage, embeddingService *EmbeddingService) *DocumentProcessor {
	return &DocumentProcessor{
		storage:          storage,
		embeddingService: embeddingService,
	}
}

// ProcessDocumentOptions contains options for processing a document
type ProcessDocumentOptions struct {
	ChunkSize     int
	ChunkOverlap  int
	ChunkStrategy ChunkingStrategy
}

// ProcessDocument processes a document: chunks it and generates embeddings
func (p *DocumentProcessor) ProcessDocument(ctx context.Context, doc *Document, opts ProcessDocumentOptions) error {
	log.Info().Str("doc_id", doc.ID).Str("title", doc.Title).Msg("Processing document")

	// Update status to processing
	if err := p.storage.UpdateDocumentStatus(ctx, doc.ID, DocumentStatusProcessing, ""); err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}

	// Delete existing chunks if reprocessing
	if err := p.storage.DeleteChunksByDocument(ctx, doc.ID); err != nil {
		log.Warn().Err(err).Str("doc_id", doc.ID).Msg("Failed to delete existing chunks")
	}

	// Set defaults
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = 512
	}
	if opts.ChunkOverlap <= 0 {
		opts.ChunkOverlap = 50
	}
	if opts.ChunkStrategy == "" {
		opts.ChunkStrategy = ChunkingStrategyRecursive
	}

	// Chunk the document
	textChunks, err := p.chunkDocument(doc.Content, opts)
	if err != nil {
		p.storage.UpdateDocumentStatus(ctx, doc.ID, DocumentStatusFailed, err.Error())
		return fmt.Errorf("failed to chunk document: %w", err)
	}

	if len(textChunks) == 0 {
		p.storage.UpdateDocumentStatus(ctx, doc.ID, DocumentStatusFailed, "No content to process")
		return fmt.Errorf("no content to process")
	}

	log.Info().Str("doc_id", doc.ID).Int("chunks", len(textChunks)).Msg("Document chunked")

	// Generate embeddings for all chunks
	embeddings, err := p.generateEmbeddings(ctx, textChunks)
	if err != nil {
		p.storage.UpdateDocumentStatus(ctx, doc.ID, DocumentStatusFailed, err.Error())
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Create chunk records
	chunks := make([]Chunk, len(textChunks))
	for i, text := range textChunks {
		tokenCount := estimateTokenCount(text)
		chunks[i] = Chunk{
			DocumentID:      doc.ID,
			KnowledgeBaseID: doc.KnowledgeBaseID,
			Content:         text,
			ChunkIndex:      i,
			TokenCount:      &tokenCount,
			Embedding:       embeddings[i],
		}
	}

	// Save chunks
	if err := p.storage.CreateChunks(ctx, chunks); err != nil {
		p.storage.UpdateDocumentStatus(ctx, doc.ID, DocumentStatusFailed, err.Error())
		return fmt.Errorf("failed to save chunks: %w", err)
	}

	// Mark document as indexed
	if err := p.storage.MarkDocumentIndexed(ctx, doc.ID); err != nil {
		return fmt.Errorf("failed to mark document indexed: %w", err)
	}

	log.Info().Str("doc_id", doc.ID).Int("chunks_created", len(chunks)).Msg("Document processing complete")
	return nil
}

// chunkDocument splits document content into chunks based on strategy
func (p *DocumentProcessor) chunkDocument(content string, opts ProcessDocumentOptions) ([]string, error) {
	// Clean content
	content = cleanText(content)

	if len(content) == 0 {
		return nil, nil
	}

	switch opts.ChunkStrategy {
	case ChunkingStrategySentence:
		return p.chunkBySentence(content, opts.ChunkSize, opts.ChunkOverlap)
	case ChunkingStrategyParagraph:
		return p.chunkByParagraph(content, opts.ChunkSize, opts.ChunkOverlap)
	case ChunkingStrategyFixed:
		return p.chunkByFixed(content, opts.ChunkSize, opts.ChunkOverlap)
	case ChunkingStrategyRecursive:
		fallthrough
	default:
		return p.chunkRecursive(content, opts.ChunkSize, opts.ChunkOverlap)
	}
}

// chunkRecursive implements recursive text splitting (most flexible)
func (p *DocumentProcessor) chunkRecursive(content string, chunkSize, overlap int) ([]string, error) {
	// Separators in order of preference
	separators := []string{"\n\n", "\n", ". ", ", ", " ", ""}

	return p.splitRecursively(content, separators, chunkSize, overlap)
}

func (p *DocumentProcessor) splitRecursively(text string, separators []string, chunkSize, overlap int) ([]string, error) {
	// Estimate characters per chunk (rough approximation: 4 chars per token)
	maxChars := chunkSize * 4

	if len(text) <= maxChars {
		return []string{strings.TrimSpace(text)}, nil
	}

	if len(separators) == 0 {
		// No more separators, split by character
		return p.splitByCharacter(text, maxChars, overlap*4)
	}

	separator := separators[0]
	remainingSeparators := separators[1:]

	parts := strings.Split(text, separator)
	if len(parts) == 1 {
		// Separator not found, try next one
		return p.splitRecursively(text, remainingSeparators, chunkSize, overlap)
	}

	var chunks []string
	var currentChunk strings.Builder
	overlapChars := overlap * 4

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if adding this part exceeds max size
		potentialSize := currentChunk.Len()
		if potentialSize > 0 {
			potentialSize += len(separator)
		}
		potentialSize += len(part)

		if potentialSize > maxChars && currentChunk.Len() > 0 {
			// Save current chunk
			chunkText := strings.TrimSpace(currentChunk.String())
			if len(chunkText) > 0 {
				chunks = append(chunks, chunkText)
			}

			// Start new chunk with overlap
			currentChunk.Reset()
			if overlapChars > 0 && len(chunkText) > overlapChars {
				overlapText := chunkText[len(chunkText)-overlapChars:]
				// Find a word boundary for overlap
				spaceIdx := strings.LastIndex(overlapText, " ")
				if spaceIdx > 0 {
					overlapText = overlapText[spaceIdx+1:]
				}
				currentChunk.WriteString(overlapText)
				if separator != "" {
					currentChunk.WriteString(separator)
				}
			}
		}

		if currentChunk.Len() > 0 && separator != "" {
			currentChunk.WriteString(separator)
		}

		// If part itself is too large, recursively split it
		if len(part) > maxChars {
			subChunks, _ := p.splitRecursively(part, remainingSeparators, chunkSize, overlap)
			for _, subChunk := range subChunks {
				chunks = append(chunks, subChunk)
			}
		} else {
			currentChunk.WriteString(part)
		}
	}

	// Don't forget the last chunk
	if currentChunk.Len() > 0 {
		chunkText := strings.TrimSpace(currentChunk.String())
		if len(chunkText) > 0 {
			chunks = append(chunks, chunkText)
		}
	}

	return chunks, nil
}

func (p *DocumentProcessor) splitByCharacter(text string, maxChars, overlapChars int) ([]string, error) {
	var chunks []string
	for i := 0; i < len(text); i += maxChars - overlapChars {
		end := i + maxChars
		if end > len(text) {
			end = len(text)
		}
		chunk := strings.TrimSpace(text[i:end])
		if len(chunk) > 0 {
			chunks = append(chunks, chunk)
		}
		if end == len(text) {
			break
		}
	}
	return chunks, nil
}

// chunkBySentence splits by sentence boundaries
func (p *DocumentProcessor) chunkBySentence(content string, chunkSize, overlap int) ([]string, error) {
	sentences := splitSentences(content)
	return p.mergeUnits(sentences, chunkSize, overlap)
}

// chunkByParagraph splits by paragraph boundaries
func (p *DocumentProcessor) chunkByParagraph(content string, chunkSize, overlap int) ([]string, error) {
	paragraphs := strings.Split(content, "\n\n")
	var cleanParagraphs []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p != "" {
			cleanParagraphs = append(cleanParagraphs, p)
		}
	}
	return p.mergeUnits(cleanParagraphs, chunkSize, overlap)
}

// chunkByFixed splits by fixed character count
func (p *DocumentProcessor) chunkByFixed(content string, chunkSize, overlap int) ([]string, error) {
	maxChars := chunkSize * 4
	overlapChars := overlap * 4
	return p.splitByCharacter(content, maxChars, overlapChars)
}

// mergeUnits merges small units (sentences/paragraphs) into chunks
func (p *DocumentProcessor) mergeUnits(units []string, chunkSize, overlap int) ([]string, error) {
	maxChars := chunkSize * 4
	overlapChars := overlap * 4

	var chunks []string
	var currentChunk strings.Builder

	for _, unit := range units {
		potentialSize := currentChunk.Len()
		if potentialSize > 0 {
			potentialSize++ // space
		}
		potentialSize += len(unit)

		if potentialSize > maxChars && currentChunk.Len() > 0 {
			chunkText := strings.TrimSpace(currentChunk.String())
			if len(chunkText) > 0 {
				chunks = append(chunks, chunkText)
			}

			currentChunk.Reset()
			// Add overlap from previous chunk
			if overlapChars > 0 && len(chunkText) > overlapChars {
				currentChunk.WriteString(chunkText[len(chunkText)-overlapChars:])
				currentChunk.WriteString(" ")
			}
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(unit)
	}

	if currentChunk.Len() > 0 {
		chunkText := strings.TrimSpace(currentChunk.String())
		if len(chunkText) > 0 {
			chunks = append(chunks, chunkText)
		}
	}

	return chunks, nil
}

// generateEmbeddings generates embeddings for all chunks
func (p *DocumentProcessor) generateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if p.embeddingService == nil {
		return nil, fmt.Errorf("embedding service not configured")
	}

	// Process in batches to avoid rate limits
	batchSize := 100
	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		resp, err := p.embeddingService.Embed(ctx, batch, "")
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings for batch %d: %w", i/batchSize, err)
		}

		allEmbeddings = append(allEmbeddings, resp.Embeddings...)
	}

	return allEmbeddings, nil
}

// splitSentences splits text into sentences
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])

		// Check for sentence end
		if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' {
			// Look ahead for space or end
			if i+1 >= len(runes) || unicode.IsSpace(runes[i+1]) {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	// Add remaining text
	if current.Len() > 0 {
		sentence := strings.TrimSpace(current.String())
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// cleanText removes extra whitespace and normalizes text
func cleanText(text string) string {
	// Replace multiple whitespace with single space
	var result strings.Builder
	var lastWasSpace bool

	for _, r := range text {
		isSpace := unicode.IsSpace(r)
		if isSpace {
			if !lastWasSpace {
				result.WriteRune(' ')
			}
			lastWasSpace = true
		} else {
			result.WriteRune(r)
			lastWasSpace = false
		}
	}

	return strings.TrimSpace(result.String())
}

// estimateTokenCount estimates the number of tokens in text
// Using rough approximation: ~4 characters per token for English
func estimateTokenCount(text string) int {
	return len(text) / 4
}

// hashContent generates SHA-256 hash of content
func hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// ProcessPendingDocuments processes all pending documents
func (p *DocumentProcessor) ProcessPendingDocuments(ctx context.Context, batchSize int) (int, error) {
	docs, err := p.storage.GetPendingDocuments(ctx, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending documents: %w", err)
	}

	processed := 0
	for _, doc := range docs {
		// Get knowledge base for config
		kb, err := p.storage.GetKnowledgeBase(ctx, doc.KnowledgeBaseID)
		if err != nil || kb == nil {
			log.Warn().Str("doc_id", doc.ID).Msg("Knowledge base not found for document")
			p.storage.UpdateDocumentStatus(ctx, doc.ID, DocumentStatusFailed, "Knowledge base not found")
			continue
		}

		opts := ProcessDocumentOptions{
			ChunkSize:     kb.ChunkSize,
			ChunkOverlap:  kb.ChunkOverlap,
			ChunkStrategy: ChunkingStrategy(kb.ChunkStrategy),
		}

		if err := p.ProcessDocument(ctx, &doc, opts); err != nil {
			log.Error().Err(err).Str("doc_id", doc.ID).Msg("Failed to process document")
			continue
		}

		processed++
	}

	return processed, nil
}

// ReprocessDocument reprocesses a document (deletes chunks and regenerates)
func (p *DocumentProcessor) ReprocessDocument(ctx context.Context, documentID string) error {
	doc, err := p.storage.GetDocument(ctx, documentID)
	if err != nil {
		return err
	}
	if doc == nil {
		return fmt.Errorf("document not found")
	}

	kb, err := p.storage.GetKnowledgeBase(ctx, doc.KnowledgeBaseID)
	if err != nil || kb == nil {
		return fmt.Errorf("knowledge base not found")
	}

	opts := ProcessDocumentOptions{
		ChunkSize:     kb.ChunkSize,
		ChunkOverlap:  kb.ChunkOverlap,
		ChunkStrategy: ChunkingStrategy(kb.ChunkStrategy),
	}

	return p.ProcessDocument(ctx, doc, opts)
}

// AddDocument adds a document to a knowledge base and processes it
func (p *DocumentProcessor) AddDocument(ctx context.Context, kbID string, req CreateDocumentRequest, userID *string) (*Document, error) {
	// Get knowledge base config
	kb, err := p.storage.GetKnowledgeBase(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, fmt.Errorf("knowledge base not found")
	}

	// Set defaults
	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = "manual"
	}

	// Create document
	doc := &Document{
		KnowledgeBaseID: kbID,
		Title:           req.Title,
		Content:         req.Content,
		SourceURL:       req.SourceURL,
		SourceType:      sourceType,
		MimeType:        req.MimeType,
		ContentHash:     hashContent(req.Content),
		Tags:            req.Tags,
		CreatedBy:       userID,
	}

	// Convert metadata
	if req.Metadata != nil {
		metadataJSON, _ := json.Marshal(req.Metadata)
		doc.Metadata = metadataJSON
	}

	if err := p.storage.CreateDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	// Process document asynchronously
	go func() {
		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		opts := ProcessDocumentOptions{
			ChunkSize:     kb.ChunkSize,
			ChunkOverlap:  kb.ChunkOverlap,
			ChunkStrategy: ChunkingStrategy(kb.ChunkStrategy),
		}

		if err := p.ProcessDocument(processCtx, doc, opts); err != nil {
			log.Error().Err(err).Str("doc_id", doc.ID).Msg("Background document processing failed")
		}
	}()

	return doc, nil
}
