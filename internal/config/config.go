package config

import (
	"os"
	"path/filepath"
)

func GetConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "lifelog"), nil
}

func GetConfigFile(name string) (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func GetRecordJSONFile() (string, error) {
	return GetConfigFile("lifelog.json")
}

func GetConfigJSONFile() (string, error) {
	return GetConfigFile("config.json")
}

func GetSqliteFile() (string, error) {
	return GetConfigFile("lifelog.db")
}
