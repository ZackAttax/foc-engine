package deviceverification

import (
	"context"
	"fmt"
	"os"

	"github.com/b-j-roberts/foc-engine/internal/config"
	"golang.org/x/oauth2/google"
)

// DeviceVerificationService manages device verification for Android and iOS
type DeviceVerificationService struct {
	GoogleCredentials   *google.Credentials
	GoogleProjectNumber string
	GooglePackageName   string
	AppleTeamID         string
	AppleKeyID          string
	ApplePrivateKeyPath string
	Initialized         bool
}

var Service *DeviceVerificationService

// InitDeviceVerification initializes the device verification service with credentials
func InitDeviceVerification() error {
	service := &DeviceVerificationService{
		Initialized: false,
	}

	// Load Google credentials
	projectNumber := config.GetGooglePlayIntegrityProjectNumber()
	packageName := config.GetGooglePlayIntegrityPackageName()
	serviceAccountJSON := config.GetGoogleCloudServiceAccountJSON()

	if projectNumber != "" && packageName != "" && serviceAccountJSON != "" {
		service.GoogleProjectNumber = projectNumber
		service.GooglePackageName = packageName

		// Load service account credentials
		var credentialsData []byte
		if _, err := os.Stat(serviceAccountJSON); err == nil {
			// It's a file path
			credentialsData, err = os.ReadFile(serviceAccountJSON)
			if err != nil {
				return fmt.Errorf("failed to read service account JSON file: %w", err)
			}
		} else {
			// Assume it's a JSON string
			credentialsData = []byte(serviceAccountJSON)
		}

		// Parse credentials for OAuth2
		credentials, err := google.CredentialsFromJSON(context.Background(), credentialsData, "https://www.googleapis.com/auth/playintegrity")
		if err != nil {
			return fmt.Errorf("failed to parse Google credentials: %w", err)
		}

		service.GoogleCredentials = credentials
	} else if projectNumber != "" || packageName != "" || serviceAccountJSON != "" {
		// Partial configuration is an error
		return fmt.Errorf("Google Play Integrity configuration incomplete: all of projectNumber, packageName, and serviceAccountJSON must be provided")
	}

	// Load Apple credentials
	appleTeamID := config.GetAppleTeamID()
	appleKeyID := config.GetAppleKeyID()
	applePrivateKeyPath := config.GetApplePrivateKeyPath()

	if appleTeamID != "" && appleKeyID != "" && applePrivateKeyPath != "" {
		service.AppleTeamID = appleTeamID
		service.AppleKeyID = appleKeyID
		service.ApplePrivateKeyPath = applePrivateKeyPath
	} else if appleTeamID != "" || appleKeyID != "" || applePrivateKeyPath != "" {
		// Partial configuration is an error
		return fmt.Errorf("Apple DeviceCheck configuration incomplete: all of teamId, keyId, and privateKeyPath must be provided")
	}

	Service = service
	service.Initialized = true

	// Log initialization status
	if service.GoogleProjectNumber != "" {
		fmt.Printf("[DeviceVerification] Google Play Integrity initialized - Project: %s, Package: %s\n",
			service.GoogleProjectNumber, service.GooglePackageName)
	} else {
		fmt.Printf("[DeviceVerification] Google Play Integrity not configured\n")
	}

	if service.AppleTeamID != "" {
		fmt.Printf("[DeviceVerification] Apple DeviceCheck initialized - TeamID: %s, KeyID: %s\n",
			service.AppleTeamID, service.AppleKeyID)
	} else {
		fmt.Printf("[DeviceVerification] Apple DeviceCheck not configured\n")
	}

	return nil
}

// IsInitialized returns whether the service has been initialized
func IsInitialized() bool {
	return Service != nil && Service.Initialized
}
