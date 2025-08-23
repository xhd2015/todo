package run

import "github.com/xhd2015/todo/data"

const DEFAULT_STORAGE = "file"

// StorageConfig holds storage-related configuration values
type StorageConfig struct {
	StorageType string
	ServerAddr  string
	ServerToken string
}

// ApplyConfigDefaults loads saved config and applies defaults to storage settings
func ApplyConfigDefaults(storageType, serverAddr, serverToken string) (StorageConfig, error) {
	// Load saved config
	savedConfig, err := data.LoadConfig()
	if err != nil {
		return StorageConfig{}, err
	}

	// Apply config defaults when command line values are not provided
	if storageType == "" && savedConfig != nil && savedConfig.StorageType != "" {
		storageType = savedConfig.StorageType
	}
	if serverAddr == "" && savedConfig != nil && savedConfig.ServerAddr != "" {
		serverAddr = savedConfig.ServerAddr
	}
	if serverToken == "" && savedConfig != nil && savedConfig.ServerToken != "" {
		serverToken = savedConfig.ServerToken
	}

	// Apply final default for storage type if still empty
	if storageType == "" {
		storageType = DEFAULT_STORAGE
	}

	return StorageConfig{
		StorageType: storageType,
		ServerAddr:  serverAddr,
		ServerToken: serverToken,
	}, nil
}
