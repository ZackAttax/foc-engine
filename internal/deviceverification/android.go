package deviceverification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DeviceRecall represents the device recall bits from Google Play Integrity
type DeviceRecall struct {
	BitFirst  bool `json:"bitFirst"`
	BitSecond bool `json:"bitSecond"`
	BitThird  bool `json:"bitThird"`
}

// AndroidIntegrityResponse represents the response from Google Play Integrity API
type AndroidIntegrityResponse struct {
	TokenPayloadExternal *TokenPayloadExternal `json:"tokenPayloadExternal"`
}

// TokenPayloadExternal contains the decoded token information
type TokenPayloadExternal struct {
	DeviceIntegrity *DeviceIntegrity `json:"deviceIntegrity"`
	DeviceRecall    *DeviceRecall    `json:"deviceRecall"`
}

// DeviceIntegrity contains device integrity information
type DeviceIntegrity struct {
	DeviceRecognitionVerdict []string `json:"deviceRecognitionVerdict"`
}

// VerifyAndroidResponse represents the response from VerifyAndroidIntegrityToken
type VerifyAndroidResponse struct {
	DeviceRecall *DeviceRecall `json:"deviceRecall"`
	Verified     bool          `json:"verified"`
	Error        string        `json:"error,omitempty"`
}

// VerifyAndroidIntegrityToken verifies an Android integrity token and returns device recall bits
func VerifyAndroidIntegrityToken(token, packageName string) (*VerifyAndroidResponse, error) {
	if Service == nil || !Service.Initialized {
		return nil, fmt.Errorf("device verification service not initialized")
	}

	if Service.GoogleProjectNumber == "" || Service.GooglePackageName == "" {
		return nil, fmt.Errorf("Google Play Integrity not configured")
	}

	// Use provided packageName or fall back to configured one
	if packageName == "" {
		packageName = Service.GooglePackageName
	}

	// Get OAuth2 token for Google API
	ctx := context.Background()
	tokenSource := Service.GoogleCredentials.TokenSource

	oauthToken, err := tokenSource.Token()
	if err != nil {
		return &VerifyAndroidResponse{
			Verified: false,
			Error:    fmt.Sprintf("failed to get OAuth token: %v", err),
		}, nil
	}

	// Build API request
	url := fmt.Sprintf("https://playintegrity.googleapis.com/v1/projects/%s/apps/%s:decodeIntegrityToken",
		Service.GoogleProjectNumber, packageName)

	requestBody := map[string]string{
		"integrityToken": token,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return &VerifyAndroidResponse{
			Verified: false,
			Error:    fmt.Sprintf("failed to marshal request: %v", err),
		}, nil
	}

	// Make HTTP request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return &VerifyAndroidResponse{
			Verified: false,
			Error:    fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oauthToken.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return &VerifyAndroidResponse{
			Verified: false,
			Error:    fmt.Sprintf("failed to call Google API: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &VerifyAndroidResponse{
			Verified: false,
			Error:    fmt.Sprintf("failed to read response: %v", err),
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &VerifyAndroidResponse{
			Verified: false,
			Error:    fmt.Sprintf("Google API returned status %d: %s", resp.StatusCode, string(body)),
		}, nil
	}

	// Parse response
	var integrityResp AndroidIntegrityResponse
	if err := json.Unmarshal(body, &integrityResp); err != nil {
		return &VerifyAndroidResponse{
			Verified: false,
			Error:    fmt.Sprintf("failed to parse response: %v", err),
		}, nil
	}

	if integrityResp.TokenPayloadExternal == nil {
		return &VerifyAndroidResponse{
			Verified: false,
			Error:    "invalid token payload",
		}, nil
	}

	// Extract device recall bits
	deviceRecall := &DeviceRecall{
		BitFirst:  false,
		BitSecond: false,
		BitThird:  false,
	}

	if integrityResp.TokenPayloadExternal.DeviceRecall != nil {
		deviceRecall = integrityResp.TokenPayloadExternal.DeviceRecall
	}

	return &VerifyAndroidResponse{
		DeviceRecall: deviceRecall,
		Verified:     true,
	}, nil
}

// MarkAndroidDeviceClaimed marks an Android device as claimed using Device Recall API
func MarkAndroidDeviceClaimed(token string) error {
	if Service == nil || !Service.Initialized {
		return fmt.Errorf("device verification service not initialized")
	}

	if Service.GoogleProjectNumber == "" || Service.GooglePackageName == "" {
		return fmt.Errorf("Google Play Integrity not configured")
	}

	// Get OAuth2 token
	ctx := context.Background()
	tokenSource := Service.GoogleCredentials.TokenSource

	oauthToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to get OAuth token: %w", err)
	}

	// Build API request
	url := fmt.Sprintf("https://playintegrity.googleapis.com/v1/projects/%s/apps/%s/deviceRecall:write",
		Service.GoogleProjectNumber, Service.GooglePackageName)

	requestBody := map[string]interface{}{
		"integrityToken": token,
		"newValues": map[string]bool{
			"bitFirst": true,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oauthToken.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Google API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Google API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
