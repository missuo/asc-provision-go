/*
 * @Author: Vincent Yang
 * @Date: 2025-05-02 01:21:47
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-05-02 01:22:31
 * @FilePath: /asc-provision-go/models/models.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */
package models

import "encoding/json"

// Device represents an Apple device
type Device struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	UDID        string `json:"udid"`
	Platform    string `json:"platform"`
	Status      string `json:"status,omitempty"`
	DeviceClass string `json:"deviceClass,omitempty"`
	Model       string `json:"model,omitempty"`
	AddedDate   string `json:"addedDate,omitempty"`
}

// Profile represents an Apple provisioning profile
type Profile struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name"`
	ProfileType    string `json:"profileType"`
	Content        string `json:"content,omitempty"`
	UUID           string `json:"uuid,omitempty"`
	CreatedDate    string `json:"createdDate,omitempty"`
	ExpirationDate string `json:"expirationDate,omitempty"`
}

// ApiRequest represents the structure for API requests
type ApiRequest struct {
	Data ApiRequestData `json:"data"`
}

// ApiRequestData represents the data part of API requests
type ApiRequestData struct {
	Type          string                 `json:"type"`
	Attributes    map[string]interface{} `json:"attributes"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
}

// ApiResponse represents the structure for API responses
type ApiResponse struct {
	Data  json.RawMessage `json:"data"`
	Links json.RawMessage `json:"links,omitempty"`
	Meta  json.RawMessage `json:"meta,omitempty"`
}
