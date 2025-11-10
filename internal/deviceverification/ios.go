package deviceverification

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/b-j-roberts/foc-engine/internal/config"
	"github.com/fxamacker/cbor/v2"
	"github.com/golang-jwt/jwt/v5"
)

// VerifyIOSResponse represents the response from VerifyIOSAttestation
type VerifyIOSResponse struct {
	HasClaimedReward bool   `json:"hasClaimedReward"`
	Verified         bool   `json:"verified"`
	Error            string `json:"error,omitempty"`
}

// GenerateChallenge generates a cryptographically random challenge (32 bytes) and returns it as base64
func GenerateChallenge() (string, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return "", fmt.Errorf("failed to generate random challenge: %w", err)
	}
	return base64.StdEncoding.EncodeToString(challenge), nil
}

// VerifyIOSAttestation verifies an iOS App Attest attestation object
func VerifyIOSAttestation(attestationObject, keyId, challenge, accountAddress string) (*VerifyIOSResponse, error) {
	if Service == nil || !Service.Initialized {
		return nil, fmt.Errorf("device verification service not initialized")
	}

	// Verify challenge matches stored challenge
	storedChallenge, err := GetChallenge(accountAddress)
	if err != nil {
		return &VerifyIOSResponse{
			Verified: false,
			Error:    fmt.Sprintf("challenge not found or expired: %v", err),
		}, nil
	}

	if storedChallenge != challenge {
		return &VerifyIOSResponse{
			Verified: false,
			Error:    "challenge mismatch",
		}, nil
	}

	// Decode CBOR attestation object
	attestationBytes, err := base64.StdEncoding.DecodeString(attestationObject)
	if err != nil {
		return &VerifyIOSResponse{
			Verified: false,
			Error:    fmt.Sprintf("failed to decode attestation: %v", err),
		}, nil
	}

	var attestation map[string]interface{}
	if err := cbor.Unmarshal(attestationBytes, &attestation); err != nil {
		return &VerifyIOSResponse{
			Verified: false,
			Error:    fmt.Sprintf("failed to parse CBOR: %v", err),
		}, nil
	}

	// Extract certificate chain from attestation
	attStmt, ok := attestation["attStmt"].(map[string]interface{})
	if !ok {
		return &VerifyIOSResponse{
			Verified: false,
			Error:    "invalid attestation structure",
		}, nil
	}

	x5c, ok := attStmt["x5c"].([]interface{})
	if !ok || len(x5c) == 0 {
		return &VerifyIOSResponse{
			Verified: false,
			Error:    "missing certificate chain",
		}, nil
	}

	// Verify certificate chain (simplified - in production, verify against Apple root CA)
	// For now, we'll just check that certificates exist and are valid
	var deviceToken string
	for i, certInterface := range x5c {
		certBytes, ok := certInterface.([]byte)
		if !ok {
			return &VerifyIOSResponse{
				Verified: false,
				Error:    fmt.Sprintf("invalid certificate format at index %d", i),
			}, nil
		}

		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			return &VerifyIOSResponse{
				Verified: false,
				Error:    fmt.Sprintf("failed to parse certificate: %v", err),
			}, nil
		}

		// First certificate is the device token we'll use for DeviceCheck
		if i == 0 {
			deviceToken = base64.StdEncoding.EncodeToString(cert.RawSubjectPublicKeyInfo)
		}
	}

	if deviceToken == "" {
		return &VerifyIOSResponse{
			Verified: false,
			Error:    "failed to extract device token",
		}, nil
	}

	// Check DeviceCheck bits to see if device has already claimed
	hasClaimed, err := CheckDeviceCheckBits(deviceToken)
	if err != nil {
		return &VerifyIOSResponse{
			Verified: false,
			Error:    fmt.Sprintf("failed to check device bits: %v", err),
		}, nil
	}

	// Clean up challenge after successful verification
	DeleteChallenge(accountAddress)

	return &VerifyIOSResponse{
		HasClaimedReward: hasClaimed,
		Verified:         true,
	}, nil
}

// CheckDeviceCheckBits queries Apple DeviceCheck API for device bits
func CheckDeviceCheckBits(deviceToken string) (bool, error) {
	if Service == nil || !Service.Initialized {
		return false, fmt.Errorf("device verification service not initialized")
	}

	if Service.AppleTeamID == "" || Service.AppleKeyID == "" || Service.ApplePrivateKeyPath == "" {
		return false, fmt.Errorf("Apple DeviceCheck not configured")
	}

	// Generate JWT token
	jwtToken, err := generateDeviceCheckJWT()
	if err != nil {
		return false, fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Generate unique transaction ID
	transactionID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Build API request
	url := "https://api.development.devicecheck.apple.com/v1/query_two_bits"
	if config.Conf.Api.Production {
		url = "https://api.devicecheck.apple.com/v1/query_two_bits"
	}

	requestBody := map[string]string{
		"device_token":   deviceToken,
		"transaction_id": transactionID,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to call DeviceCheck API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("DeviceCheck API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var deviceCheckResp struct {
		Bit0           int    `json:"bit0"`
		Bit1           int    `json:"bit1"`
		LastUpdateTime string `json:"last_update_time"`
	}

	if err := json.Unmarshal(body, &deviceCheckResp); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	// bit0 represents if device has claimed reward
	return deviceCheckResp.Bit0 == 1, nil
}

// MarkIOSDeviceClaimed marks an iOS device as claimed using DeviceCheck API
func MarkIOSDeviceClaimed(deviceToken string) error {
	if Service == nil || !Service.Initialized {
		return fmt.Errorf("device verification service not initialized")
	}

	if Service.AppleTeamID == "" || Service.AppleKeyID == "" || Service.ApplePrivateKeyPath == "" {
		return fmt.Errorf("Apple DeviceCheck not configured")
	}

	// Generate JWT token
	jwtToken, err := generateDeviceCheckJWT()
	if err != nil {
		return fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Generate unique transaction ID
	transactionID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Build API request
	url := "https://api.development.devicecheck.apple.com/v1/update_two_bits"
	if config.Conf.Api.Production {
		url = "https://api.devicecheck.apple.com/v1/update_two_bits"
	}

	requestBody := map[string]interface{}{
		"device_token":   deviceToken,
		"transaction_id": transactionID,
		"bit0":           1, // Mark as claimed
		"bit1":           0,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call DeviceCheck API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("DeviceCheck API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// generateDeviceCheckJWT generates a JWT token for Apple DeviceCheck API authentication
func generateDeviceCheckJWT() (string, error) {
	if Service == nil {
		return "", fmt.Errorf("service not initialized")
	}

	// Load private key
	keyData, err := os.ReadFile(Service.ApplePrivateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS1 format
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse private key: %w", err)
		}
	}

	ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("private key is not ECDSA")
	}

	// Create JWT token
	now := time.Now().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": Service.AppleTeamID,
		"iat": now,
	})

	token.Header["kid"] = Service.AppleKeyID

	tokenString, err := token.SignedString(ecdsaKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
