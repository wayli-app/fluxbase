package ai

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/kapmahc/epub"
	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/rs/zerolog/log"
	"github.com/xuri/excelize/v2"
)

// TextExtractor extracts text content from various document formats
type TextExtractor struct {
	ocrService *OCRService
}

// NewTextExtractor creates a new text extractor without OCR support
func NewTextExtractor() *TextExtractor {
	return &TextExtractor{}
}

// NewTextExtractorWithOCR creates a new text extractor with OCR fallback support
func NewTextExtractorWithOCR(ocrService *OCRService) *TextExtractor {
	return &TextExtractor{
		ocrService: ocrService,
	}
}

// SupportedMimeTypes returns the list of MIME types supported by the extractor
func (e *TextExtractor) SupportedMimeTypes() []string {
	return []string{
		"application/pdf",
		"text/plain",
		"text/markdown",
		"text/html",
		"text/csv",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/rtf",
		"application/epub+zip",
		"application/json",
	}
}

// Extract extracts text from a document based on its MIME type
func (e *TextExtractor) Extract(data []byte, mimeType string) (string, error) {
	return e.ExtractWithLanguages(data, mimeType, nil)
}

// ExtractWithLanguages extracts text from a document with optional OCR language hints
func (e *TextExtractor) ExtractWithLanguages(data []byte, mimeType string, languages []string) (string, error) {
	var text string
	var err error

	switch mimeType {
	case "application/pdf":
		text, err = e.ExtractFromPDFWithLanguages(data, languages)
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		text, err = e.ExtractFromDOCX(data)
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		text, err = e.ExtractFromXLSX(data)
	case "text/html":
		text, err = e.ExtractFromHTML(data)
	case "text/csv":
		text, err = e.ExtractFromCSV(data)
	case "application/rtf":
		text, err = e.ExtractFromRTF(data)
	case "application/epub+zip":
		text, err = e.ExtractFromEPUB(data)
	case "text/plain", "text/markdown", "application/json":
		text, err = e.ExtractFromText(data)
	default:
		return "", fmt.Errorf("unsupported MIME type: %s", mimeType)
	}

	if err != nil {
		return "", err
	}

	// Sanitize extracted text to remove null bytes and other invalid UTF-8 sequences
	// that would cause PostgreSQL errors
	return SanitizeText(text), nil
}

// SanitizeText removes null bytes and other characters that are invalid in PostgreSQL UTF-8 text
func SanitizeText(text string) string {
	// Remove null bytes (0x00) which PostgreSQL rejects
	text = strings.ReplaceAll(text, "\x00", "")

	// Also remove other problematic control characters (except newline, tab, carriage return)
	var builder strings.Builder
	builder.Grow(len(text))

	for _, r := range text {
		// Allow printable characters, spaces, tabs, newlines, and carriage returns
		if r == '\t' || r == '\n' || r == '\r' || (r >= 0x20 && r != 0x7F) {
			builder.WriteRune(r)
		}
		// Skip control characters (0x00-0x1F except tab/newline/cr, and 0x7F)
	}

	return builder.String()
}

// ExtractFromPDF extracts text from PDF documents
func (e *TextExtractor) ExtractFromPDF(data []byte) (string, error) {
	return e.ExtractFromPDFWithLanguages(data, nil)
}

// ExtractFromPDFWithLanguages extracts text from PDF documents with OCR fallback
// If standard text extraction returns garbage/binary data, it falls back to OCR
func (e *TextExtractor) ExtractFromPDFWithLanguages(data []byte, languages []string) (string, error) {
	// First, try standard PDF text extraction
	text, err := e.extractPDFText(data)
	if err != nil {
		// If PDF parsing fails completely, try OCR if available
		if e.ocrService != nil && e.ocrService.IsEnabled() {
			log.Debug().Err(err).Msg("Standard PDF parsing failed, attempting OCR")
			return e.extractPDFWithOCR(data, languages)
		}
		return "", err
	}

	// Check if extracted text is valid (not garbage/binary data)
	if IsValidTextContent(text) {
		return text, nil
	}

	// Text quality is poor - try OCR if available
	if e.ocrService != nil && e.ocrService.IsEnabled() {
		log.Debug().
			Int("original_length", len(text)).
			Float64("quality_score", TextQualityScore(text)).
			Msg("Standard PDF extraction returned poor quality text, attempting OCR")

		ocrText, ocrErr := e.extractPDFWithOCR(data, languages)
		if ocrErr == nil && IsValidTextContent(ocrText) {
			return ocrText, nil
		}
		if ocrErr != nil {
			log.Warn().Err(ocrErr).Msg("OCR extraction also failed")
		}
	}

	// Return original text if OCR not available or failed
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("no text could be extracted from PDF (document may be image-based and OCR is not available)")
	}

	// Return the original text even if quality is poor (better than nothing)
	return text, nil
}

// extractPDFText is the original PDF text extraction logic
func (e *TextExtractor) extractPDFText(data []byte) (string, error) {
	reader := bytes.NewReader(data)
	pdfReader, err := pdf.NewReader(reader, int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to read PDF: %w", err)
	}

	var text strings.Builder
	numPages := pdfReader.NumPage()

	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		content, err := page.GetPlainText(nil)
		if err != nil {
			// Continue with other pages even if one fails
			continue
		}
		text.WriteString(content)
		text.WriteString("\n\n")
	}

	return strings.TrimSpace(text.String()), nil
}

// extractPDFWithOCR uses the OCR service to extract text from a PDF
func (e *TextExtractor) extractPDFWithOCR(data []byte, languages []string) (string, error) {
	if e.ocrService == nil || !e.ocrService.IsEnabled() {
		return "", fmt.Errorf("OCR service not available")
	}

	result, err := e.ocrService.ExtractTextFromPDF(context.Background(), data, languages)
	if err != nil {
		return "", err
	}

	log.Info().
		Int("pages", result.Pages).
		Float64("confidence", result.Confidence).
		Int("text_length", len(result.Text)).
		Str("language", result.Language).
		Msg("OCR extraction completed successfully")

	return result.Text, nil
}

// ExtractFromDOCX extracts text from Word documents
func (e *TextExtractor) ExtractFromDOCX(data []byte) (string, error) {
	reader := bytes.NewReader(data)
	doc, err := docx.ReadDocxFromMemory(reader, int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to read DOCX: %w", err)
	}
	defer doc.Close()

	content := doc.Editable().GetContent()
	return strings.TrimSpace(content), nil
}

// ExtractFromXLSX extracts text from Excel spreadsheets
func (e *TextExtractor) ExtractFromXLSX(data []byte) (string, error) {
	reader := bytes.NewReader(data)
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read XLSX: %w", err)
	}
	defer f.Close()

	var text strings.Builder
	sheets := f.GetSheetList()

	for _, sheet := range sheets {
		text.WriteString(fmt.Sprintf("=== Sheet: %s ===\n", sheet))

		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}

		for _, row := range rows {
			text.WriteString(strings.Join(row, "\t"))
			text.WriteString("\n")
		}
		text.WriteString("\n")
	}

	return strings.TrimSpace(text.String()), nil
}

// ExtractFromHTML extracts text from HTML documents by stripping tags
func (e *TextExtractor) ExtractFromHTML(data []byte) (string, error) {
	content := string(data)

	// Remove script and style tags with their content
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	content = scriptRe.ReplaceAllString(content, "")

	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	content = styleRe.ReplaceAllString(content, "")

	// Replace block elements with newlines
	blockRe := regexp.MustCompile(`(?i)</(div|p|h[1-6]|li|tr|br|hr)[^>]*>`)
	content = blockRe.ReplaceAllString(content, "\n")

	// Remove all remaining HTML tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	content = tagRe.ReplaceAllString(content, "")

	// Decode common HTML entities
	content = strings.ReplaceAll(content, "&nbsp;", " ")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&#39;", "'")

	// Clean up whitespace
	spaceRe := regexp.MustCompile(`[ \t]+`)
	content = spaceRe.ReplaceAllString(content, " ")

	newlineRe := regexp.MustCompile(`\n{3,}`)
	content = newlineRe.ReplaceAllString(content, "\n\n")

	return strings.TrimSpace(content), nil
}

// ExtractFromCSV extracts text from CSV files
func (e *TextExtractor) ExtractFromCSV(data []byte) (string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	var text strings.Builder
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Try to continue despite errors
			continue
		}
		text.WriteString(strings.Join(record, "\t"))
		text.WriteString("\n")
	}

	return strings.TrimSpace(text.String()), nil
}

// ExtractFromRTF extracts text from RTF documents
func (e *TextExtractor) ExtractFromRTF(data []byte) (string, error) {
	content := string(data)

	// Remove RTF control groups
	groupRe := regexp.MustCompile(`\{[^{}]*\}`)
	for groupRe.MatchString(content) {
		content = groupRe.ReplaceAllString(content, "")
	}

	// Remove RTF control words
	controlRe := regexp.MustCompile(`\\[a-z]+\d*\s?`)
	content = controlRe.ReplaceAllString(content, "")

	// Remove backslash escapes
	content = strings.ReplaceAll(content, "\\{", "{")
	content = strings.ReplaceAll(content, "\\}", "}")
	content = strings.ReplaceAll(content, "\\\\", "\\")

	// Clean up
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	spaceRe := regexp.MustCompile(`[ \t]+`)
	content = spaceRe.ReplaceAllString(content, " ")

	newlineRe := regexp.MustCompile(`\n{3,}`)
	content = newlineRe.ReplaceAllString(content, "\n\n")

	return strings.TrimSpace(content), nil
}

// ExtractFromEPUB extracts text from EPUB e-books
func (e *TextExtractor) ExtractFromEPUB(data []byte) (string, error) {
	// Write data to a temp file since epub library needs a file path
	tmpFile, err := os.CreateTemp("", "epub-*.epub")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	book, err := epub.Open(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read EPUB: %w", err)
	}
	defer book.Close()

	var text strings.Builder

	// Extract text from each chapter
	for _, rf := range book.Opf.Manifest {
		if rf.MediaType == "application/xhtml+xml" || rf.MediaType == "text/html" {
			content, err := book.Open(rf.Href)
			if err != nil {
				continue
			}
			contentBytes, err := io.ReadAll(content)
			content.Close()
			if err != nil {
				continue
			}

			// Extract text from HTML content
			extracted, err := e.ExtractFromHTML(contentBytes)
			if err == nil && extracted != "" {
				text.WriteString(extracted)
				text.WriteString("\n\n")
			}
		}
	}

	return strings.TrimSpace(text.String()), nil
}

// ExtractFromText handles plain text, markdown, and JSON files
func (e *TextExtractor) ExtractFromText(data []byte) (string, error) {
	return string(data), nil
}

// GetMimeTypeFromExtension returns the MIME type for a file extension
func GetMimeTypeFromExtension(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	mimeTypes := map[string]string{
		"pdf":  "application/pdf",
		"txt":  "text/plain",
		"md":   "text/markdown",
		"html": "text/html",
		"htm":  "text/html",
		"csv":  "text/csv",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"rtf":  "application/rtf",
		"epub": "application/epub+zip",
		"json": "application/json",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// GetExtensionFromMimeType returns the file extension for a MIME type
func GetExtensionFromMimeType(mimeType string) string {
	extensions := map[string]string{
		"application/pdf": ".pdf",
		"text/plain":      ".txt",
		"text/markdown":   ".md",
		"text/html":       ".html",
		"text/csv":        ".csv",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       ".xlsx",
		"application/rtf":      ".rtf",
		"application/epub+zip": ".epub",
		"application/json":     ".json",
	}

	if ext, ok := extensions[mimeType]; ok {
		return ext
	}
	return ""
}
