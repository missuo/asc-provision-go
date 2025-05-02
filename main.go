package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
)

// Config represents application configuration
type Config struct {
	KeyID          string
	IssuerID       string
	PrivateKeyPath string
	BundleID       string
	APIBaseURL     string
}

// JWTGenerator handles JWT token generation
type JWTGenerator struct {
	Config     Config
	PrivateKey *ecdsa.PrivateKey
}

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

// LoadConfig loads configuration from environment variables
func LoadConfig() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	config := Config{
		KeyID:          os.Getenv("APPLE_KEY_ID"),
		IssuerID:       os.Getenv("APPLE_ISSUER_ID"),
		PrivateKeyPath: os.Getenv("APPLE_PRIVATE_KEY_PATH"),
		BundleID:       os.Getenv("APPLE_BUNDLE_ID"),
		APIBaseURL:     "https://api.appstoreconnect.apple.com/v1",
	}

	if config.KeyID == "" || config.IssuerID == "" || config.PrivateKeyPath == "" || config.BundleID == "" {
		return config, fmt.Errorf("missing required environment variables")
	}

	return config, nil
}

// NewJWTGenerator initializes a JWT generator
func NewJWTGenerator(config Config) (*JWTGenerator, error) {
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

// ApiClient handles API requests to Apple's API
type ApiClient struct {
	Config       Config
	JWTGenerator *JWTGenerator
	Client       *http.Client
}

// NewApiClient creates a new API client
func NewApiClient(config Config, jwtGenerator *JWTGenerator) *ApiClient {
	return &ApiClient{
		Config:       config,
		JWTGenerator: jwtGenerator,
		Client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// SendRequest sends an API request to Apple's API
func (c *ApiClient) SendRequest(method, endpoint string, queryParams map[string]string, body interface{}) ([]byte, error) {
	url := c.Config.APIBaseURL + endpoint

	// Add query parameters
	if len(queryParams) > 0 {
		urlParams := "?"
		i := 0
		for key, value := range queryParams {
			if i > 0 {
				urlParams += "&"
			}
			urlParams += key + "=" + value
			i++
		}
		url += urlParams
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Generate a new JWT token
	token, err := c.JWTGenerator.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(respBody))
	}

	return respBody, nil
}

// GetDevices fetches all devices with optional platform filter
func (c *ApiClient) GetDevices(platformFilter string) ([]Device, error) {
	queryParams := map[string]string{
		"limit": "200",
	}

	if platformFilter != "" {
		queryParams["filter[platform]"] = platformFilter
	}

	respBody, err := c.SendRequest("GET", "/devices", queryParams, nil)
	if err != nil {
		return nil, err
	}

	var apiResp ApiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var deviceList struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				AddedDate   string `json:"addedDate"`
				Name        string `json:"name"`
				DeviceClass string `json:"deviceClass"`
				Model       string `json:"model"`
				UDID        string `json:"udid"`
				Platform    string `json:"platform"`
				Status      string `json:"status"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &deviceList); err != nil {
		return nil, fmt.Errorf("failed to parse device list: %w", err)
	}

	devices := make([]Device, len(deviceList.Data))
	for i, d := range deviceList.Data {
		devices[i] = Device{
			ID:          d.ID,
			Name:        d.Attributes.Name,
			UDID:        d.Attributes.UDID,
			Platform:    d.Attributes.Platform,
			Status:      d.Attributes.Status,
			DeviceClass: d.Attributes.DeviceClass,
			Model:       d.Attributes.Model,
			AddedDate:   d.Attributes.AddedDate,
		}
	}

	return devices, nil
}

// RegisterDevice registers a new device
func (c *ApiClient) RegisterDevice(name, udid, platform string) (*Device, error) {
	requestData := ApiRequest{
		Data: ApiRequestData{
			Type: "devices",
			Attributes: map[string]interface{}{
				"name":     name,
				"udid":     udid,
				"platform": platform,
			},
		},
	}

	respBody, err := c.SendRequest("POST", "/devices", nil, requestData)
	if err != nil {
		return nil, err
	}

	var deviceResp struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				AddedDate   string `json:"addedDate"`
				Name        string `json:"name"`
				DeviceClass string `json:"deviceClass"`
				Model       string `json:"model"`
				UDID        string `json:"udid"`
				Platform    string `json:"platform"`
				Status      string `json:"status"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &deviceResp); err != nil {
		return nil, fmt.Errorf("failed to parse device response: %w", err)
	}

	return &Device{
		ID:          deviceResp.Data.ID,
		Name:        deviceResp.Data.Attributes.Name,
		UDID:        deviceResp.Data.Attributes.UDID,
		Platform:    deviceResp.Data.Attributes.Platform,
		Status:      deviceResp.Data.Attributes.Status,
		DeviceClass: deviceResp.Data.Attributes.DeviceClass,
		Model:       deviceResp.Data.Attributes.Model,
		AddedDate:   deviceResp.Data.Attributes.AddedDate,
	}, nil
}

// GetProfiles fetches all provisioning profiles
func (c *ApiClient) GetProfiles() ([]Profile, error) {
	respBody, err := c.SendRequest("GET", "/profiles", map[string]string{"limit": "200"}, nil)
	if err != nil {
		return nil, err
	}

	var profileList struct {
		Data []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name           string `json:"name"`
				ProfileType    string `json:"profileType"`
				Content        string `json:"profileContent"`
				UUID           string `json:"uuid"`
				CreatedDate    string `json:"createdDate"`
				ExpirationDate string `json:"expirationDate"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &profileList); err != nil {
		return nil, fmt.Errorf("failed to parse profile list: %w", err)
	}

	profiles := make([]Profile, len(profileList.Data))
	for i, p := range profileList.Data {
		profiles[i] = Profile{
			ID:             p.ID,
			Name:           p.Attributes.Name,
			ProfileType:    p.Attributes.ProfileType,
			Content:        p.Attributes.Content,
			UUID:           p.Attributes.UUID,
			CreatedDate:    p.Attributes.CreatedDate,
			ExpirationDate: p.Attributes.ExpirationDate,
		}
	}

	return profiles, nil
}

// GetProfile fetches a specific profile by ID
func (c *ApiClient) GetProfile(profileID string) (*Profile, error) {
	respBody, err := c.SendRequest("GET", "/profiles/"+profileID, nil, nil)
	if err != nil {
		return nil, err
	}

	var profileResp struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name           string `json:"name"`
				ProfileType    string `json:"profileType"`
				Content        string `json:"profileContent"`
				UUID           string `json:"uuid"`
				CreatedDate    string `json:"createdDate"`
				ExpirationDate string `json:"expirationDate"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &profileResp); err != nil {
		return nil, fmt.Errorf("failed to parse profile response: %w", err)
	}

	return &Profile{
		ID:             profileResp.Data.ID,
		Name:           profileResp.Data.Attributes.Name,
		ProfileType:    profileResp.Data.Attributes.ProfileType,
		Content:        profileResp.Data.Attributes.Content,
		UUID:           profileResp.Data.Attributes.UUID,
		CreatedDate:    profileResp.Data.Attributes.CreatedDate,
		ExpirationDate: profileResp.Data.Attributes.ExpirationDate,
	}, nil
}

// DeleteProfile deletes a profile by ID
func (c *ApiClient) DeleteProfile(profileID string) error {
	_, err := c.SendRequest("DELETE", "/profiles/"+profileID, nil, nil)
	return err
}

// GetBundleIDInfo retrieves the internal ID for a bundle identifier
func (c *ApiClient) GetBundleIDInfo(bundleID string) (string, error) {
	queryParams := map[string]string{
		"filter[identifier]": bundleID,
	}

	respBody, err := c.SendRequest("GET", "/bundleIds", queryParams, nil)
	if err != nil {
		return "", err
	}

	var bundleResp struct {
		Data []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &bundleResp); err != nil {
		return "", fmt.Errorf("failed to parse bundle ID response: %w", err)
	}

	if len(bundleResp.Data) == 0 {
		return "", fmt.Errorf("bundle ID not found: %s", bundleID)
	}

	return bundleResp.Data[0].ID, nil
}

// GetDistributionCertificates fetches iOS distribution certificates
func (c *ApiClient) GetDistributionCertificates() (string, error) {
	queryParams := map[string]string{
		"filter[certificateType]": "IOS_DISTRIBUTION",
	}

	respBody, err := c.SendRequest("GET", "/certificates", queryParams, nil)
	if err != nil {
		return "", err
	}

	var certResp struct {
		Data []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &certResp); err != nil {
		return "", fmt.Errorf("failed to parse certificate response: %w", err)
	}

	if len(certResp.Data) == 0 {
		return "", fmt.Errorf("no distribution certificates found")
	}

	return certResp.Data[0].ID, nil
}

// CreateProfile creates a new profile
func (c *ApiClient) CreateProfile(name, profileType string) (*Profile, error) {
	// Get Bundle ID information
	log.Printf("Getting Bundle ID info for: %s", c.Config.BundleID)
	bundleIDRef, err := c.GetBundleIDInfo(c.Config.BundleID)
	if err != nil {
		log.Printf("ERROR getting Bundle ID info: %v", err)
		return nil, err
	}
	log.Printf("Got Bundle ID reference: %s", bundleIDRef)

	// Get certificate information
	log.Printf("Getting distribution certificates")
	certID, err := c.GetDistributionCertificates()
	log.Printf("Got certificate ID: %s", certID)
	if err != nil {
		log.Printf("ERROR getting distribution certificates: %v", err)
		return nil, err
	}

	// Get all iOS devices
	log.Printf("Getting all iOS devices")
	devices, err := c.GetDevices("IOS")
	if err != nil {
		log.Printf("ERROR getting iOS devices: %v", err)
		return nil, err
	}
	log.Printf("Got %d iOS devices", len(devices))

	// 过滤设备以匹配配置文件类型
	var filteredDevices []Device

	// 根据配置文件类型过滤设备类型
	if profileType == "IOS_APP_ADHOC" || profileType == "IOS_APP_DEVELOPMENT" || profileType == "IOS_APP_STORE" {
		// 只包含iPhone、iPod、iPad类型的设备
		for _, device := range devices {
			// 如果设备是iPhone，或deviceClass为空
			if device.DeviceClass == "IPHONE" || device.DeviceClass == "IPOD" || device.DeviceClass == "IPAD" || device.DeviceClass == "" {
				// 确保平台是iOS
				if device.Platform == "IOS" {
					filteredDevices = append(filteredDevices, device)
				}
			}
		}
	} else if profileType == "MAC_APP_DEVELOPMENT" || profileType == "MAC_APP_STORE" || profileType == "MAC_APP_DIRECT" {
		// 只包含Mac类型的设备
		for _, device := range devices {
			if device.Platform == "MAC_OS" {
				filteredDevices = append(filteredDevices, device)
			}
		}
	}

	log.Printf("Filtered to %d compatible devices", len(filteredDevices))

	// 如果没有兼容设备，返回错误
	if len(filteredDevices) == 0 {
		return nil, fmt.Errorf("no compatible devices found for profile type: %s", profileType)
	}

	// Prepare device IDs list
	log.Printf("Preparing device data for request")
	deviceData := make([]map[string]string, len(filteredDevices))
	for i, device := range filteredDevices {
		deviceData[i] = map[string]string{
			"id":   device.ID,
			"type": "devices",
		}
	}

	// Build request data
	log.Printf("Building request data")
	requestData := ApiRequest{
		Data: ApiRequestData{
			Type: "profiles",
			Attributes: map[string]interface{}{
				"name":        name,
				"profileType": profileType,
			},
			Relationships: map[string]interface{}{
				"bundleId": map[string]interface{}{
					"data": map[string]string{
						"id":   bundleIDRef,
						"type": "bundleIds",
					},
				},
				"certificates": map[string]interface{}{
					"data": []map[string]string{
						{
							"id":   certID,
							"type": "certificates",
						},
					},
				},
				"devices": map[string]interface{}{
					"data": deviceData,
				},
			},
		},
	}

	// 发送请求
	log.Printf("Sending POST request to /profiles")
	respBody, err := c.SendRequest("POST", "/profiles", nil, requestData)
	if err != nil {
		log.Printf("ERROR sending request: %v", err)
		return nil, err
	}
	log.Printf("Received response from profiles endpoint")

	// Parse response
	log.Printf("Parsing response")
	var profileResp struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name           string `json:"name"`
				ProfileType    string `json:"profileType"`
				Content        string `json:"profileContent"`
				UUID           string `json:"uuid"`
				CreatedDate    string `json:"createdDate"`
				ExpirationDate string `json:"expirationDate"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &profileResp); err != nil {
		log.Printf("ERROR parsing response: %v", err)
		return nil, fmt.Errorf("failed to parse profile creation response: %w", err)
	}

	// 创建结果profile
	result := &Profile{
		ID:             profileResp.Data.ID,
		Name:           profileResp.Data.Attributes.Name,
		ProfileType:    profileResp.Data.Attributes.ProfileType,
		Content:        profileResp.Data.Attributes.Content,
		UUID:           profileResp.Data.Attributes.UUID,
		CreatedDate:    profileResp.Data.Attributes.CreatedDate,
		ExpirationDate: profileResp.Data.Attributes.ExpirationDate,
	}

	log.Printf("PROFILE CREATION SUCCESSFUL")
	log.Printf("Profile ID: %s", result.ID)
	log.Printf("Profile Name: %s", result.Name)

	return result, nil
}

// DownloadProfile downloads and saves a profile to disk
func (c *ApiClient) DownloadProfile(profileID, outputDir string) (string, error) {
	profile, err := c.GetProfile(profileID)
	if err != nil {
		return "", err
	}

	if profile.Content == "" {
		return "", fmt.Errorf("profile has no content")
	}

	// Decode Base64 content
	content, err := base64.StdEncoding.DecodeString(profile.Content)
	if err != nil {
		return "", fmt.Errorf("failed to decode profile content: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create a safe filename
	safeName := strings.ReplaceAll(profile.Name, " ", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	filename := filepath.Join(outputDir, safeName+".mobileprovision")

	// Write to file
	if err := os.WriteFile(filename, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write profile to file: %w", err)
	}

	return filename, nil
}

// CreateAndDownloadAdHocProfile creates and downloads an Ad Hoc profile
func (c *ApiClient) CreateAndDownloadAdHocProfile(outputDir string) (string, error) {
	// Create profile name with timestamp
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	profileName := "OwO_" + timestamp

	// Create profile
	profile, err := c.CreateProfile(profileName, "IOS_APP_ADHOC")
	if err != nil {
		return "", err
	}

	// Download profile
	return c.DownloadProfile(profile.ID, outputDir)
}

// Controller handles HTTP requests
type Controller struct {
	ApiClient *ApiClient
}

// NewController creates a new controller
func NewController(apiClient *ApiClient) *Controller {
	return &Controller{
		ApiClient: apiClient,
	}
}

// GetDevices handles device listing endpoint
func (c *Controller) GetDevices(ctx *gin.Context) {
	platform := ctx.DefaultQuery("platform", "IOS")

	devices, err := c.ApiClient.GetDevices(platform)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"devices": devices})
}

// RegisterDevice handles device registration endpoint
func (c *Controller) RegisterDevice(ctx *gin.Context) {
	var device Device
	if err := ctx.ShouldBindJSON(&device); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if device.Name == "" || device.UDID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Name and UDID are required"})
		return
	}

	if device.Platform == "" {
		device.Platform = "IOS"
	}

	newDevice, err := c.ApiClient.RegisterDevice(device.Name, device.UDID, device.Platform)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"device": newDevice})
}

// GetProfiles handles profile listing endpoint
func (c *Controller) GetProfiles(ctx *gin.Context) {
	profiles, err := c.ApiClient.GetProfiles()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"profiles": profiles})
}

// GetProfile handles single profile retrieval endpoint
func (c *Controller) GetProfile(ctx *gin.Context) {
	profileID := ctx.Param("id")
	if profileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Profile ID is required"})
		return
	}

	profile, err := c.ApiClient.GetProfile(profileID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"profile": profile})
}

// DeleteProfile handles profile deletion endpoint
func (c *Controller) DeleteProfile(ctx *gin.Context) {
	profileID := ctx.Param("id")
	if profileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Profile ID is required"})
		return
	}

	err := c.ApiClient.DeleteProfile(profileID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Profile deleted successfully"})
}

// CreateProfile handles profile creation endpoint
func (c *Controller) CreateProfile(ctx *gin.Context) {
	var profile Profile
	if err := ctx.ShouldBindJSON(&profile); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if profile.Name == "" {
		// Use default name with timestamp
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		profile.Name = "OwO_" + timestamp
	}

	if profile.ProfileType == "" {
		profile.ProfileType = "IOS_APP_ADHOC"
	}

	newProfile, err := c.ApiClient.CreateProfile(profile.Name, profile.ProfileType)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"profile": newProfile})
}

// DownloadProfile handles profile download endpoint
func (c *Controller) DownloadProfile(ctx *gin.Context) {
	profileID := ctx.Param("id")
	if profileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Profile ID is required"})
		return
	}

	// Use temp directory
	outputDir := os.TempDir()
	filePath, err := c.ApiClient.DownloadProfile(profileID, outputDir)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get filename
	_, filename := filepath.Split(filePath)

	// Set response headers and provide file download
	ctx.FileAttachment(filePath, filename)

	// Delete temp file after download
	go func() {
		time.Sleep(5 * time.Second)
		os.Remove(filePath)
	}()
}

// CreateAndDownloadProfile handles combined create and download endpoint
func (c *Controller) CreateAndDownloadProfile(ctx *gin.Context) {
	// Use temp directory
	outputDir := os.TempDir()

	filePath, err := c.ApiClient.CreateAndDownloadAdHocProfile(outputDir)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get filename
	_, filename := filepath.Split(filePath)

	// Set response headers and provide file download
	ctx.FileAttachment(filePath, filename)

	// Delete temp file after download
	go func() {
		time.Sleep(5 * time.Second)
		os.Remove(filePath)
	}()
}

func main() {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize JWT generator
	jwtGenerator, err := NewJWTGenerator(config)
	if err != nil {
		log.Fatalf("Failed to initialize JWT generator: %v", err)
	}

	// Create API client
	apiClient := NewApiClient(config, jwtGenerator)

	// Create controller
	controller := NewController(apiClient)

	// Set up Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

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

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
