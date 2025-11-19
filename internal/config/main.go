package config

func Load() {
	loadFromEnv()
	loadFromConfig()
}
