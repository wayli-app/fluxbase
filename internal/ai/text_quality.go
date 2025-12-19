package ai

import (
	"regexp"
	"strings"
	"unicode"
)

// IsValidTextContent checks if extracted text is meaningful content.
// Returns true if the text appears to be valid readable text, false if it's garbage/binary data.
// This is used to detect when PDF text extraction fails and falls back to OCR.
func IsValidTextContent(text string) bool {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return false
	}

	totalRunes := 0
	printableCount := 0
	controlCount := 0
	letterCount := 0

	for _, r := range text {
		totalRunes++

		// Count printable characters (includes letters, numbers, punctuation, symbols)
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			printableCount++
		}

		// Count control characters (excluding common whitespace)
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			controlCount++
		}

		// Count letters (any script)
		if unicode.IsLetter(r) {
			letterCount++
		}
	}

	if totalRunes == 0 {
		return false
	}

	// Check 1: Any control characters is a bad sign (should be 0 for valid text)
	// Even a few control characters indicate binary/garbage data
	if controlCount > 0 {
		controlRatio := float64(controlCount) / float64(totalRunes)
		if controlRatio > 0.02 { // More than 2% control chars = garbage
			return false
		}
	}

	// Check 2: Printable character ratio (should be >90%)
	printableRatio := float64(printableCount) / float64(totalRunes)
	if printableRatio < 0.90 {
		return false
	}

	// For very short text, the above checks are sufficient
	if len(text) < 20 {
		return true
	}

	// Check 3: Word-like pattern ratio (for longer text)
	// Check for sequences of 2+ letters (words in any language)
	wordPattern := regexp.MustCompile(`\p{L}{2,}`)
	words := wordPattern.FindAllString(text, -1)
	wordLetterCount := 0
	for _, w := range words {
		wordLetterCount += len([]rune(w))
	}

	// At least 30% of characters should be in word-like patterns
	wordRatio := float64(wordLetterCount) / float64(totalRunes)
	return wordRatio >= 0.30
}

// TextQualityScore returns a score from 0-1 indicating text quality.
// Higher scores indicate more readable text.
func TextQualityScore(text string) float64 {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return 0
	}

	totalRunes := 0
	printableCount := 0
	letterCount := 0

	for _, r := range text {
		totalRunes++
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			printableCount++
		}
		if unicode.IsLetter(r) {
			letterCount++
		}
	}

	if totalRunes == 0 {
		return 0
	}

	printableRatio := float64(printableCount) / float64(totalRunes)
	letterRatio := float64(letterCount) / float64(totalRunes)

	// Weighted average: 60% printable ratio, 40% letter ratio
	return printableRatio*0.6 + letterRatio*0.4
}
