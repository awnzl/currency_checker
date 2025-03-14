package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	ReqTimeout int // seconds
	RateLimit  int // seconds
	RetryNum   int
}

func InitConfig(configPath string) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	viper.SetDefault("request.timeout", 5)
	viper.BindEnv("request.timeout", "REQUEST_TIMEOUT")

	viper.SetDefault("request.rate_limit", 30)
	viper.BindEnv("request.rate_limit", "REQUEST_RATE_LIMIT")

	viper.SetDefault("request.retry_num", 5)
	viper.BindEnv("request.retry_num", "REQUEST_RETRY_NUM")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file: %v", err)
	}
}

func GetConfig() Config {
	return Config{
		ReqTimeout: viper.GetInt("request.timeout"),
		RateLimit:  viper.GetInt("request.rate_limit"),
		RetryNum:   viper.GetInt("request.retry_num"),
	}
}