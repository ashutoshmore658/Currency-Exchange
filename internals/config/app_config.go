package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort       string `mapstructure:"SERVER_PORT"`
	ExternalAPIURL   string `mapstructure:"EXTERNAL_API_URL"`
	HistoryDaysLimit int    `mapstructure:"HISTORY_DAYS_LIMIT"`
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("EXTERNAL_API_URL", "https://api.exchangerate.host")
	viper.SetDefault("HISTORY_DAYS_LIMIT", 90)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	cfg.ServerPort = viper.GetString("SERVER_PORT")
	cfg.ExternalAPIURL = viper.GetString("EXTERNAL_API_URL")

	log.Printf("Configuration loaded: %+v", cfg)
	return &cfg, nil
}
