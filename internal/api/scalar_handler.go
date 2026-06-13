package api

import (
	"github.com/gofiber/fiber/v3"
)

const scalarHTML = `<!DOCTYPE html>
<html>
  <head>
    <title>Fluxbase API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/openapi.json"
      data-proxy-url=""
    ></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`

func HandleScalarAPIReference(c fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("Cache-Control", "no-cache")
	return c.SendString(scalarHTML)
}
