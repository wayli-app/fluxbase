package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// renderMagicLinkHTML Tests
// =============================================================================

func TestRenderMagicLinkHTML(t *testing.T) {
	t.Run("default template", func(t *testing.T) {
		html := renderMagicLinkHTML("https://example.com/login?token=abc123", "abc123", "")

		assert.Contains(t, html, "https://example.com/login?token=abc123")
		assert.Contains(t, html, "abc123")
		assert.Contains(t, html, "<html")
		assert.Contains(t, html, "</html>")
	})

	t.Run("with custom template", func(t *testing.T) {
		// Create temp template file
		tmpDir := t.TempDir()
		templatePath := filepath.Join(tmpDir, "magic-link.html")
		templateContent := `<html><body><h1>Custom Magic Link</h1><p>Link: {{.Link}}</p><p>Token: {{.Token}}</p></body></html>`
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		html := renderMagicLinkHTML("https://example.com/login", "token123", templatePath)

		assert.Contains(t, html, "Custom Magic Link")
		assert.Contains(t, html, "https://example.com/login")
		assert.Contains(t, html, "token123")
	})

	t.Run("nonexistent custom template falls back to default", func(t *testing.T) {
		html := renderMagicLinkHTML("https://example.com/login", "token", "/nonexistent/path.html")

		// Should use default template
		assert.Contains(t, html, "https://example.com/login")
		assert.Contains(t, html, "<html")
	})

	t.Run("empty link and token", func(t *testing.T) {
		html := renderMagicLinkHTML("", "", "")

		// Should still render without errors
		assert.Contains(t, html, "<html")
	})

	t.Run("link with special characters", func(t *testing.T) {
		link := "https://example.com/login?token=abc&user=test@example.com"
		html := renderMagicLinkHTML(link, "abc", "")

		assert.Contains(t, html, "example.com")
	})
}

// =============================================================================
// renderVerificationHTML Tests
// =============================================================================

func TestRenderVerificationHTML(t *testing.T) {
	t.Run("default template", func(t *testing.T) {
		html := renderVerificationHTML("https://example.com/verify?token=xyz", "xyz", "")

		assert.Contains(t, html, "https://example.com/verify?token=xyz")
		assert.Contains(t, html, "xyz")
		assert.Contains(t, html, "<html")
		assert.Contains(t, html, "</html>")
	})

	t.Run("with custom template", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatePath := filepath.Join(tmpDir, "verify.html")
		templateContent := `<html><body><h1>Custom Verification</h1><a href="{{.Link}}">Verify Now</a><p>Code: {{.Token}}</p></body></html>`
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		html := renderVerificationHTML("https://example.com/verify", "code456", templatePath)

		assert.Contains(t, html, "Custom Verification")
		assert.Contains(t, html, "https://example.com/verify")
		assert.Contains(t, html, "code456")
	})

	t.Run("nonexistent custom template falls back to default", func(t *testing.T) {
		html := renderVerificationHTML("https://example.com/verify", "token", "/bad/path.html")

		assert.Contains(t, html, "https://example.com/verify")
		assert.Contains(t, html, "<html")
	})
}

// =============================================================================
// renderPasswordResetHTML Tests
// =============================================================================

func TestRenderPasswordResetHTML(t *testing.T) {
	t.Run("default template", func(t *testing.T) {
		html := renderPasswordResetHTML("https://example.com/reset?token=reset123", "reset123", "")

		assert.Contains(t, html, "https://example.com/reset?token=reset123")
		assert.Contains(t, html, "reset123")
		assert.Contains(t, html, "<html")
		assert.Contains(t, html, "</html>")
	})

	t.Run("with custom template", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatePath := filepath.Join(tmpDir, "reset.html")
		templateContent := `<html><body><h1>Reset Password</h1><p>Click <a href="{{.Link}}">here</a> to reset. Token: {{.Token}}</p></body></html>`
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		html := renderPasswordResetHTML("https://example.com/reset", "resetToken", templatePath)

		assert.Contains(t, html, "Reset Password")
		assert.Contains(t, html, "https://example.com/reset")
		assert.Contains(t, html, "resetToken")
	})

	t.Run("invalid template syntax falls back to default", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatePath := filepath.Join(tmpDir, "bad.html")
		// Invalid template syntax
		templateContent := `<html>{{.InvalidSyntax`
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		html := renderPasswordResetHTML("https://example.com/reset", "token", templatePath)

		// Should fall back to default (or at least not panic)
		assert.Contains(t, html, "<html")
	})
}

// =============================================================================
// renderInvitationHTML Tests
// =============================================================================

func TestRenderInvitationHTML(t *testing.T) {
	t.Run("with inviter name", func(t *testing.T) {
		html := renderInvitationHTML("John Doe", "https://example.com/invite/abc")

		assert.Contains(t, html, "John Doe")
		assert.Contains(t, html, "https://example.com/invite/abc")
		assert.Contains(t, html, "<html")
	})

	t.Run("empty inviter name", func(t *testing.T) {
		html := renderInvitationHTML("", "https://example.com/invite/xyz")

		assert.Contains(t, html, "https://example.com/invite/xyz")
		assert.Contains(t, html, "<html")
	})

	t.Run("special characters in inviter name", func(t *testing.T) {
		html := renderInvitationHTML("Test <User>", "https://example.com/invite")

		// Should handle or escape special characters
		assert.Contains(t, html, "<html")
	})
}

// =============================================================================
// loadAndRenderTemplate Tests
// =============================================================================

func TestLoadAndRenderTemplate(t *testing.T) {
	t.Run("valid template", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatePath := filepath.Join(tmpDir, "test.html")
		templateContent := `<html><body>Hello {{.Name}}!</body></html>`
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		data := map[string]string{"Name": "World"}
		html := loadAndRenderTemplate(templatePath, data)

		assert.Contains(t, html, "Hello World!")
	})

	t.Run("nonexistent file returns empty", func(t *testing.T) {
		data := map[string]string{"Key": "Value"}
		html := loadAndRenderTemplate("/nonexistent/path/template.html", data)

		assert.Empty(t, html)
	})

	t.Run("invalid template syntax returns empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatePath := filepath.Join(tmpDir, "invalid.html")
		templateContent := `<html>{{.Broken`
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		data := map[string]string{"Key": "Value"}
		html := loadAndRenderTemplate(templatePath, data)

		assert.Empty(t, html)
	})

	t.Run("template with missing variable", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatePath := filepath.Join(tmpDir, "missing.html")
		templateContent := `<html>{{.Missing}}</html>`
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		data := map[string]string{"Other": "Value"}
		html := loadAndRenderTemplate(templatePath, data)

		// Should render with empty value for missing key
		assert.Contains(t, html, "<html>")
	})

	t.Run("empty data map", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatePath := filepath.Join(tmpDir, "nodata.html")
		templateContent := `<html><body>Static content</body></html>`
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		html := loadAndRenderTemplate(templatePath, map[string]string{})

		assert.Contains(t, html, "Static content")
	})

	t.Run("nil data map", func(t *testing.T) {
		tmpDir := t.TempDir()
		templatePath := filepath.Join(tmpDir, "nildata.html")
		templateContent := `<html><body>No data needed</body></html>`
		err := os.WriteFile(templatePath, []byte(templateContent), 0644)
		require.NoError(t, err)

		html := loadAndRenderTemplate(templatePath, nil)

		assert.Contains(t, html, "No data needed")
	})
}

// =============================================================================
// Fallback HTML Functions Tests
// =============================================================================

func TestFallbackMagicLinkHTML(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		contains []string
	}{
		{
			name:     "basic link",
			link:     "https://example.com/login",
			contains: []string{"https://example.com/login", "<html>", "</html>", "Login", "href"},
		},
		{
			name:     "empty link",
			link:     "",
			contains: []string{"<html>", "</html>", "href"},
		},
		{
			name:     "link with query params",
			link:     "https://example.com/login?token=abc&redirect=/dashboard",
			contains: []string{"example.com", "<html>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := fallbackMagicLinkHTML(tt.link)

			for _, expected := range tt.contains {
				assert.Contains(t, html, expected)
			}
		})
	}
}

func TestFallbackVerificationHTML(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		contains []string
	}{
		{
			name:     "basic link",
			link:     "https://example.com/verify",
			contains: []string{"https://example.com/verify", "<html>", "</html>", "Verify", "href"},
		},
		{
			name:     "empty link",
			link:     "",
			contains: []string{"<html>", "</html>", "href"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := fallbackVerificationHTML(tt.link)

			for _, expected := range tt.contains {
				assert.Contains(t, html, expected)
			}
		})
	}
}

func TestFallbackPasswordResetHTML(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		contains []string
	}{
		{
			name:     "basic link",
			link:     "https://example.com/reset",
			contains: []string{"https://example.com/reset", "<html>", "</html>", "Reset", "href"},
		},
		{
			name:     "empty link",
			link:     "",
			contains: []string{"<html>", "</html>", "href"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := fallbackPasswordResetHTML(tt.link)

			for _, expected := range tt.contains {
				assert.Contains(t, html, expected)
			}
		})
	}
}

func TestFallbackInvitationHTML(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		contains []string
	}{
		{
			name:     "basic link",
			link:     "https://example.com/invite/abc",
			contains: []string{"https://example.com/invite/abc", "<html>", "</html>", "Invited", "href"},
		},
		{
			name:     "empty link",
			link:     "",
			contains: []string{"<html>", "</html>", "href"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := fallbackInvitationHTML(tt.link)

			for _, expected := range tt.contains {
				assert.Contains(t, html, expected)
			}
		})
	}
}

// =============================================================================
// Template Security Tests
// =============================================================================

func TestTemplateHTMLEscaping(t *testing.T) {
	t.Run("magic link escapes HTML in token", func(t *testing.T) {
		// Token with potential XSS payload
		html := renderMagicLinkHTML("https://example.com", "<script>alert('xss')</script>", "")

		// Should not contain unescaped script tag
		assert.NotContains(t, html, "<script>alert")
	})

	t.Run("invitation escapes HTML in inviter name", func(t *testing.T) {
		// Inviter name with HTML
		html := renderInvitationHTML("<script>alert('xss')</script>", "https://example.com")

		// HTML template package should escape this
		assert.NotContains(t, html, "<script>alert")
	})
}

// =============================================================================
// Template Output Validation Tests
// =============================================================================

func TestTemplateOutputsValidHTML(t *testing.T) {
	templates := []struct {
		name string
		html string
	}{
		{"magic link", renderMagicLinkHTML("https://example.com", "token", "")},
		{"verification", renderVerificationHTML("https://example.com", "token", "")},
		{"password reset", renderPasswordResetHTML("https://example.com", "token", "")},
		{"invitation", renderInvitationHTML("Inviter", "https://example.com")},
		{"fallback magic link", fallbackMagicLinkHTML("https://example.com")},
		{"fallback verification", fallbackVerificationHTML("https://example.com")},
		{"fallback password reset", fallbackPasswordResetHTML("https://example.com")},
		{"fallback invitation", fallbackInvitationHTML("https://example.com")},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			// Check basic HTML structure
			assert.True(t, strings.Contains(tt.html, "<html") || strings.Contains(tt.html, "<HTML"))
			assert.True(t, strings.Contains(tt.html, "</html>") || strings.Contains(tt.html, "</HTML>"))
			assert.Contains(t, tt.html, "<body")
			assert.Contains(t, tt.html, "</body>")
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkRenderMagicLinkHTML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = renderMagicLinkHTML("https://example.com/login?token=benchmark", "benchmark", "")
	}
}

func BenchmarkRenderVerificationHTML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = renderVerificationHTML("https://example.com/verify?token=benchmark", "benchmark", "")
	}
}

func BenchmarkRenderPasswordResetHTML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = renderPasswordResetHTML("https://example.com/reset?token=benchmark", "benchmark", "")
	}
}

func BenchmarkRenderInvitationHTML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = renderInvitationHTML("Benchmark User", "https://example.com/invite/benchmark")
	}
}

func BenchmarkLoadAndRenderTemplate(b *testing.B) {
	// Create temp template for benchmark
	tmpDir := b.TempDir()
	templatePath := filepath.Join(tmpDir, "bench.html")
	templateContent := `<html><body>{{.Link}} - {{.Token}}</body></html>`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		b.Fatal(err)
	}
	data := map[string]string{"Link": "https://example.com", "Token": "token123"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = loadAndRenderTemplate(templatePath, data)
	}
}

func BenchmarkFallbackMagicLinkHTML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = fallbackMagicLinkHTML("https://example.com/login")
	}
}
