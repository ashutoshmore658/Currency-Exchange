package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort         string        `mapstructure:"SERVER_PORT"`
	ExternalAPIURL     string        `mapstructure:"EXTERNAL_API_URL"`
	LatestRateCacheTTL time.Duration `mapstructure:"LATEST_RATE_CACHE_TTL"`
	HistoricalCacheTTL time.Duration `mapstructure:"HISTORICAL_CACHE_TTL"`
	RefreshInterval    time.Duration `mapstructure:"REFRESH_INTERVAL"`
	HistoryDaysLimit   int           `mapstructure:"HISTORY_DAYS_LIMIT"`
	RedisAddr          string        `mapstructure:"REDIS_ADDR"`
	RedisPassword      string        `mapstructure:"REDIS_PASSWORD"`
	RedisDB            int           `mapstructure:"REDIS_DB"`
	DateFmt            string        `mapstructure:"DATE_FMT"`
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("EXTERNAL_API_URL", "https://api.frankfurter.app/")
	viper.SetDefault("LATEST_RATE_CACHE_TTL", "55m")
	viper.SetDefault("HISTORICAL_CACHE_TTL", "24h")
	viper.SetDefault("REFRESH_INTERVAL", "1h")
	viper.SetDefault("HISTORY_DAYS_LIMIT", 90)

	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("DATE_FMT", "2006-01-02")

	viper.AutomaticEnv()

	cfg := &Config{}
	cfg.ServerPort = viper.GetString("SERVER_PORT")
	cfg.ExternalAPIURL = viper.GetString("EXTERNAL_API_URL")
	cfg.DateFmt = viper.GetString("DATE_FMT")
	cfg.LatestRateCacheTTL, _ = time.ParseDuration(viper.GetString("LATEST_RATE_CACHE_TTL"))
	cfg.HistoricalCacheTTL, _ = time.ParseDuration(viper.GetString("HISTORICAL_CACHE_TTL"))
	cfg.RefreshInterval, _ = time.ParseDuration(viper.GetString("REFRESH_INTERVAL"))
	cfg.HistoryDaysLimit = viper.GetInt("HISTORY_DAYS_LIMIT")

	cfg.RedisAddr = viper.GetString("REDIS_ADDR")
	cfg.RedisPassword = viper.GetString("REDIS_PASSWORD")
	cfg.RedisDB = viper.GetInt("REDIS_DB")

	log.Printf("Config loaded: %+v", cfg)
	return cfg, nil
}
