package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/hiratazx/projecthorizon/internal/api"
)

var Version = "dev"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Set release mode in production
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// API routes
	apiGroup := router.Group("/api")
	{
		// Public routes
		api.RegisterAuthRoutes(apiGroup)

		// Protected routes (auth middleware applied inside)
		api.RegisterSystemRoutes(apiGroup)
		api.RegisterDockerRoutes(apiGroup)
		api.RegisterStorageRoutes(apiGroup)
		api.RegisterSettingsRoutes(apiGroup)
	}

	// Serve static files - try multiple locations
	staticPaths := []string{
		"./public",
		"/var/lib/projecthorizon/www",
		filepath.Join(filepath.Dir(os.Args[0]), "public"),
	}

	var staticPath string
	for _, p := range staticPaths {
		if _, err := os.Stat(p); err == nil {
			staticPath = p
			break
		}
	}

	if staticPath != "" {
		router.Static("/css", filepath.Join(staticPath, "css"))
		router.Static("/js", filepath.Join(staticPath, "js"))
		router.StaticFile("/", filepath.Join(staticPath, "index.html"))
		router.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(staticPath, "index.html"))
		})
	} else {
		router.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "ProjectHorizon API Server - Static files not found")
		})
	}

	// Print banner
	printBanner(port)

	// Start server
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func printBanner(port string) {
	banner := `
╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║   ██████╗ ██████╗  ██████╗      ██╗███████╗ ██████╗████████╗  ║
║   ██╔══██╗██╔══██╗██╔═══██╗     ██║██╔════╝██╔════╝╚══██╔══╝  ║
║   ██████╔╝██████╔╝██║   ██║     ██║█████╗  ██║        ██║     ║
║   ██╔═══╝ ██╔══██╗██║   ██║██   ██║██╔══╝  ██║        ██║     ║
║   ██║     ██║  ██║╚██████╔╝╚█████╔╝███████╗╚██████╗   ██║     ║
║   ╚═╝     ╚═╝  ╚═╝ ╚═════╝  ╚════╝ ╚══════╝ ╚═════╝   ╚═╝     ║
║                     H O R I Z O N                             ║
║                                                               ║
╠═══════════════════════════════════════════════════════════════╣
║   Version: %-10s                                         ║
║   Dashboard: http://localhost:%-6s                          ║
╚═══════════════════════════════════════════════════════════════╝
`
	fmt.Printf(banner, Version, port)
}
