package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/missuo/asc-provision-go/config"
	"github.com/missuo/asc-provision-go/models"
)

// ApiClient handles API requests to Apple's API
type ApiClient struct {
	Config       *config.Config
	JWTGenerator *JWTGenerator
	Client       *http.Client
}

// NewApiClient creates a new API client
func NewApiClient(config *config.Config, jwtGenerator *JWTGenerator) *ApiClient {
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
func (c *ApiClient) GetDevices(platformFilter string) ([]models.Device, error) {
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

	var apiResp models.ApiResponse
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

	devices := make([]models.Device, len(deviceList.Data))
	for i, d := range deviceList.Data {
		devices[i] = models.Device{
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
func (c *ApiClient) RegisterDevice(name, udid, platform string) (*models.Device, error) {
	requestData := models.ApiRequest{
		Data: models.ApiRequestData{
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

	return &models.Device{
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
func (c *ApiClient) GetProfiles() ([]models.Profile, error) {
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

	profiles := make([]models.Profile, len(profileList.Data))
	for i, p := range profileList.Data {
		profiles[i] = models.Profile{
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
func (c *ApiClient) GetProfile(profileID string) (*models.Profile, error) {
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

	return &models.Profile{
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
func (c *ApiClient) CreateProfile(name, profileType string) (*models.Profile, error) {
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

	// Filter devices to match profile type
	var filteredDevices []models.Device

	// Filter devices based on profile type
	if profileType == "IOS_APP_ADHOC" || profileType == "IOS_APP_DEVELOPMENT" || profileType == "IOS_APP_STORE" {
		// Only include iPhone, iPod, iPad type devices
		for _, device := range devices {
			// If device is iPhone, or deviceClass is empty
			if device.DeviceClass == "IPHONE" || device.DeviceClass == "IPOD" || device.DeviceClass == "IPAD" || device.DeviceClass == "" {
				// Ensure platform is iOS
				if device.Platform == "IOS" {
					filteredDevices = append(filteredDevices, device)
				}
			}
		}
	} else if profileType == "MAC_APP_DEVELOPMENT" || profileType == "MAC_APP_STORE" || profileType == "MAC_APP_DIRECT" {
		// Only include Mac type devices
		for _, device := range devices {
			if device.Platform == "MAC_OS" {
				filteredDevices = append(filteredDevices, device)
			}
		}
	}

	log.Printf("Filtered to %d compatible devices", len(filteredDevices))

	// If no compatible devices, return error
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
	requestData := models.ApiRequest{
		Data: models.ApiRequestData{
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

	// Send request
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

	// Create result profile
	result := &models.Profile{
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
