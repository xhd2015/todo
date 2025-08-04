package data

import (
	"encoding/json"
	"os"

	"github.com/xhd2015/todo/internal/config"
	"github.com/xhd2015/todo/models"
)

func LoadConfig() (*models.Config, error) {
	configFile, err := config.GetConfigJSONFile()
	if err != nil {
		return nil, err
	}

	configData, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if len(configData) == 0 {
		return nil, nil
	}

	var config models.Config
	err = json.Unmarshal(configData, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func SaveConfig(conf *models.Config) error {
	configFile, err := config.GetConfigJSONFile()
	if err != nil {
		return err
	}

	data, err := json.Marshal(conf)
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}
