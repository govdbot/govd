package config

func Load() {
	loadEnv()
	loadExtractorConfigs()
}
