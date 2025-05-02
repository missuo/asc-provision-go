/*
 * @Author: Vincent Yang
 * @Date: 2025-05-02 00:49:14
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-05-02 01:27:04
 * @FilePath: /asc-provision-go/main.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */
package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/missuo/asc-provision-go/api"
	"github.com/missuo/asc-provision-go/config"
	"github.com/missuo/asc-provision-go/handlers"
	"github.com/missuo/asc-provision-go/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize JWT generator
	jwtGenerator, err := api.NewJWTGenerator(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize JWT generator: %v", err)
	}

	// Create API client
	apiClient := api.NewApiClient(cfg, jwtGenerator)

	// Create controller
	controller := handlers.NewController(apiClient)

	// Set Release Mode
	gin.SetMode(gin.ReleaseMode)

	// Set up Gin router
	r := gin.Default()

	// Add API key authentication middleware
	r.Use(middleware.APIKeyAuth(cfg))

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, API-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint (not protected by API key)
	r.GET("/health", controller.HealthCheck)

	// API routes
	api := r.Group("/api")
	{
		// Device management
		api.GET("/devices", controller.GetDevices)
		api.POST("/devices", controller.RegisterDevice)

		// Profile management
		api.GET("/profiles", controller.GetProfiles)
		api.GET("/profiles/:id", controller.GetProfile)
		api.DELETE("/profiles/:id", controller.DeleteProfile)
		api.POST("/profiles", controller.CreateProfile)
		api.GET("/profiles/:id/download", controller.DownloadProfile)
		api.POST("/profiles/create-and-download", controller.CreateAndDownloadProfile)
	}

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
