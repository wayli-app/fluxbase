package adminui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
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

	// Serve admin UI at /admin
	app.Use("/admin", filesystem.New(filesystem.Config{
		Root:         http.FS(distFS),
		PathPrefix:   "",
		Browse:       false,
		Index:        "index.html",
		NotFoundFile: "index.html",
	}))
}
