package adminui

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

//go:embed all:dist
var adminUIFiles embed.FS

// Handler serves the embedded admin UI
type Handler struct{}

// New creates a new admin UI handler
func New() *Handler {
	return &Handler{}
}

// RegisterRoutes registers admin UI routes
func (h *Handler) RegisterRoutes(app *fiber.App) {
	// Get the dist subdirectory from the embedded filesystem
	distFS, err := fs.Sub(adminUIFiles, "dist")
	if err != nil {
		panic(err)
	}

	// Create HTTP filesystem for serving
	httpFS := http.FS(distFS)

	// Custom handler for admin UI with proper SPA routing
	app.Use("/admin", func(c *fiber.Ctx) error {
		// Get the path relative to /admin
		path := strings.TrimPrefix(c.Path(), "/admin")
		if path == "" {
			path = "/"
		}

		// Try to open the file from the embedded filesystem
		file, err := httpFS.Open(path)
		if err == nil {
			defer file.Close()

			// Get file info for headers
			stat, err := file.Stat()
			if err == nil && !stat.IsDir() {
				// Set appropriate headers
				c.Set("Cache-Control", "public, max-age=3600")

				// Detect content type based on file extension
				contentType := getContentType(path)
				c.Set("Content-Type", contentType)

				// Serve the file
				content, err := io.ReadAll(file)
				if err == nil {
					return c.Send(content)
				}
			}
		}

		// If file not found or is a directory, serve index.html for SPA routing
		indexFile, err := httpFS.Open("/index.html")
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Not Found")
		}
		defer indexFile.Close()

		c.Set("Content-Type", "text/html")
		c.Set("Cache-Control", "no-cache")

		content, err := io.ReadAll(indexFile)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		return c.Send(content)
	})
}

// getContentType returns the appropriate content type for a file path
func getContentType(path string) string {
	if strings.HasSuffix(path, ".js") {
		return "application/javascript"
	}
	if strings.HasSuffix(path, ".css") {
		return "text/css"
	}
	if strings.HasSuffix(path, ".json") {
		return "application/json"
	}
	if strings.HasSuffix(path, ".png") {
		return "image/png"
	}
	if strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(path, ".svg") {
		return "image/svg+xml"
	}
	if strings.HasSuffix(path, ".woff") {
		return "font/woff"
	}
	if strings.HasSuffix(path, ".woff2") {
		return "font/woff2"
	}
	if strings.HasSuffix(path, ".ttf") {
		return "font/ttf"
	}
	if strings.HasSuffix(path, ".html") {
		return "text/html"
	}
	return "application/octet-stream"
}
