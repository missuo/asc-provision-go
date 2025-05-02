/*
 * @Author: Vincent Yang
 * @Date: 2025-05-02 01:22:01
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-05-02 01:24:49
 * @FilePath: /asc-provision-go/api/jwt.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */
package api

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/missuo/asc-provision-go/config"
)

// JWTGenerator handles JWT token generation
type JWTGenerator struct {
	Config     *config.Config
	PrivateKey *ecdsa.PrivateKey
}

// NewJWTGenerator initializes a JWT generator
func NewJWTGenerator(config *config.Config) (*JWTGenerator, error) {
	privateKeyData, err := os.ReadFile(config.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse PEM formatted private key
	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA private key")
	}

	return &JWTGenerator{
		Config:     config,
		PrivateKey: ecdsaKey,
	}, nil
}

// Generate creates a new JWT token
func (g *JWTGenerator) Generate() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": g.Config.IssuerID,
		"exp": now.Add(20 * time.Minute).Unix(),
		"aud": "appstoreconnect-v1",
		"iat": now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = g.Config.KeyID

	tokenString, err := token.SignedString(g.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
