/*
 * @Author: Vincent Yang
 * @Date: 2025-05-02 01:21:31
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-05-02 01:22:21
 * @FilePath: /asc-provision-go/config/config.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */
package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config represents application configuration
type Config struct {
	KeyID          string
	IssuerID       string
	PrivateKeyPath string
	BundleID       string
	APIBaseURL     string
	APIKey         string
	Port           string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	config := &Config{
		KeyID:          os.Getenv("APPLE_KEY_ID"),
		IssuerID:       os.Getenv("APPLE_ISSUER_ID"),
		PrivateKeyPath: os.Getenv("APPLE_PRIVATE_KEY_PATH"),
		BundleID:       os.Getenv("APPLE_BUNDLE_ID"),
		APIBaseURL:     "https://api.appstoreconnect.apple.com/v1",
		APIKey:         os.Getenv("API_KEY"),
		Port:           os.Getenv("PORT"),
	}

	if config.KeyID == "" || config.IssuerID == "" || config.PrivateKeyPath == "" || config.BundleID == "" {
		return nil, fmt.Errorf("missing required Apple API environment variables")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("missing API_KEY environment variable")
	}

	if config.Port == "" {
		config.Port = "8080"
	}

	return config, nil
}
