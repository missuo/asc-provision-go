package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/missuo/asc-provision-go/api"
	"github.com/missuo/asc-provision-go/models"
)

// Controller handles HTTP requests
type Controller struct {
	ApiClient *api.ApiClient
}

// NewController creates a new controller
func NewController(apiClient *api.ApiClient) *Controller {
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
	var device models.Device
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
	var profile models.Profile
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

// HealthCheck handles the health check endpoint
func (c *Controller) HealthCheck(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}
