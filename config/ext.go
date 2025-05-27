package config

import (
	"fmt"
	"maps"
	"os"

	"github.com/govdbot/govd/models"

	"gopkg.in/yaml.v3"
)

const configPath = "config.yaml"

var extractorConfigs map[string]*models.ExtractorConfig

func LoadExtractorConfigs() error {
	extractorConfigs = make(map[string]*models.ExtractorConfig)

	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed reading config file: %w", err)
	}

	var rawConfig map[string]*models.ExtractorConfig

	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed parsing config file: %w", err)
	}
	maps.Copy(extractorConfigs, rawConfig)

	return nil
}

func GetExtractorConfig(extractor *models.Extractor) *models.ExtractorConfig {
	if config, exists := extractorConfigs[extractor.CodeName]; exists {
		return config
	}
	return nil
}
