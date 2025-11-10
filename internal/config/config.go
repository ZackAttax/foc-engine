package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RpcConfig struct {
	Host string `yaml:"Host"`
}

type ApiConfig struct {
	Host         string   `yaml:"Host"`
	Port         int      `yaml:"Port"`
	AllowOrigins []string `yaml:"AllowOrigins"`
	AllowMethods []string `yaml:"AllowMethods"`
	AllowHeaders []string `yaml:"AllowHeaders"`
	Production   bool     `yaml:"Production"`
	Admin        bool     `yaml:"Admin"`
}

type IndexerConfig struct {
	Host    string `yaml:"Host"`
	Port    int    `yaml:"Port"`
	StartAt *int   `yaml:"StartAt,omitempty"` // Optional field, can be nil
}

type PaymasterConfig struct {
	Network    string `yaml:"Network"`
	ApiUrl     string `yaml:"ApiUrl"`
}

type Config struct {
	Rpc       RpcConfig       `yaml:"Rpc"`
	Api       ApiConfig       `yaml:"Api"`
	Indexer   IndexerConfig   `yaml:"Indexer"`
	Paymaster PaymasterConfig `yaml:"Paymaster"`
	Modules   []string        `yaml:"Modules"`
}

var Conf *Config

func InitConfig() {
	configPath, ok := os.LookupEnv("CONFIG_PATH")
	if !ok {
		configPath = "configs/config.yaml"
		fmt.Println("CONFIG_PATH not set, using default config.yaml")
	}

	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Println("Error reading config file: ", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(yamlFile, &Conf)
	if err != nil {
		fmt.Println("Error parsing config file: ", err)
		os.Exit(1)
	}
}

// Modules enum
type FocModule string

const (
	ModulePaymaster FocModule = "AVNU_PAYMASTER"
	ModuleAccounts  FocModule = "ACCOUNTS"
	ModuleEvents    FocModule = "EVENTS"
	ModuleRegistry  FocModule = "REGISTRY"
)

func ModuleEnabled(module FocModule) bool {
	if Conf == nil {
		return false
	}
	for _, m := range Conf.Modules {
		if m == string(module) {
			return true
		}
	}
	return false
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(envKey, defaultValue string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	return defaultValue
}

// GetPaymasterNetwork returns the network to use for paymaster, with fallback logic
func GetPaymasterNetwork() string {
	// Priority: environment variable > config file > default
	if network := os.Getenv("PAYMASTER_NETWORK"); network != "" {
		return network
	}
	if Conf != nil && Conf.Paymaster.Network != "" {
		return Conf.Paymaster.Network
	}
	return "sepolia"
}

// GetPaymasterApiUrl returns the paymaster API URL based on network
func GetPaymasterApiUrl() string {
	// Priority: environment variable > config file > network-based default
	if apiUrl := os.Getenv("PAYMASTER_API_URL"); apiUrl != "" {
		return apiUrl
	}
	if Conf != nil && Conf.Paymaster.ApiUrl != "" {
		return Conf.Paymaster.ApiUrl
	}
	
	// Network-based defaults
	network := GetPaymasterNetwork()
	switch network {
	case "mainnet":
		return "https://starknet.api.avnu.fi"
	case "sepolia":
		return "https://sepolia.api.avnu.fi"
	default:
		return "https://sepolia.api.avnu.fi"
	}
}

// GetPaymasterApiKey returns the API key for the paymaster service
func GetPaymasterApiKey() string {
  // Check if environment variable is set
  if key := os.Getenv("AVNU_PAYMASTER_API_KEY"); key != "" {
    return key
  } else {
    fmt.Println("Warning: AVNU_PAYMASTER_API_KEY environment variable is not set. Proceeding without an API key.")
    return ""
  }
}

// GetGooglePlayIntegrityProjectNumber returns the Google Play Integrity project number
func GetGooglePlayIntegrityProjectNumber() string {
	return getEnvOrDefault("GOOGLE_PLAY_INTEGRITY_PROJECT_NUMBER", "")
}

// GetGooglePlayIntegrityPackageName returns the Google Play Integrity package name
func GetGooglePlayIntegrityPackageName() string {
	return getEnvOrDefault("GOOGLE_PLAY_INTEGRITY_PACKAGE_NAME", "")
}

// GetGoogleCloudServiceAccountJSON returns the Google Cloud service account JSON (file path or JSON string)
func GetGoogleCloudServiceAccountJSON() string {
	return getEnvOrDefault("GOOGLE_CLOUD_SERVICE_ACCOUNT_JSON", "")
}

// GetAppleTeamID returns the Apple Team ID for DeviceCheck
func GetAppleTeamID() string {
	return getEnvOrDefault("APPLE_TEAM_ID", "")
}

// GetAppleKeyID returns the Apple Key ID for DeviceCheck
func GetAppleKeyID() string {
	return getEnvOrDefault("APPLE_KEY_ID", "")
}

// GetApplePrivateKeyPath returns the path to the Apple private key file for DeviceCheck
func GetApplePrivateKeyPath() string {
	return getEnvOrDefault("APPLE_PRIVATE_KEY_PATH", "")
}
