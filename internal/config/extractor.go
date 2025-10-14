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

	validateConfig()
}

func validateConfig() {
	for id, cfg := range extractorConfigs {
		var active int
		if cfg.Proxy != "" {
			active++
		}
		if cfg.EdgeProxy != "" {
			active++
		}
		if cfg.DisableProxy {
			active++
		}
		if active > 1 {
			logger.L.Fatalf("[%s] invalid config: cannot enable more than one proxy option at the same time", id)
		}
		if cfg.Instance != "" && id != "youtube" {
			logger.L.Fatalf("[%s] invalid config: custom instance is only supported for youtube extractor", id)
		}
	}
}

func GetExtractorConfig(extractorID string) *ExtractorConfig {
	if config, exists := extractorConfigs[extractorID]; exists {
		return config
	}
	return &ExtractorConfig{}
}
