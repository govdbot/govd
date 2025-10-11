package config

import (
	"maps"
	"os"

	"github.com/govdbot/govd/internal/logger"
	"gopkg.in/yaml.v2"
)

const configPath = "private/config.yaml"

var extractorConfigs map[string]*ExtractorConfig

func loadExtractorConfigs() {
	extractorConfigs = make(map[string]*ExtractorConfig)

	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.L.Fatalf("failed reading config file: %v", err)
	}

	var rawConfig map[string]*ExtractorConfig

	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		logger.L.Fatalf("failed parsing config file: %v", err)
	}
	maps.Copy(extractorConfigs, rawConfig)
}

func GetExtractorConfig(extractorID string) *ExtractorConfig {
	if config, exists := extractorConfigs[extractorID]; exists {
		return config
	}
	return nil
}
