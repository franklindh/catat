package util

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DBDriver                string        `mapstructure:"DB_DRIVER"`
	DBSource                string        `mapstructure:"DB_SOURCE"`
	ServerAddress           string        `mapstructure:"SERVER_ADDRESS"`
	TokenSymmetricKey       string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration     time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	GoogleOAuthClientID     string        `mapstructure:"GOOGLE_OAUTH_CLIENT_ID"`
	GoogleOAuthClientSecret string        `mapstructure:"GOOGLE_OAUTH_CLIENT_SECRET"`
	GoogleOAuthRedirectURL  string        `mapstructure:"GOOGLE_OAUTH_REDIRECT_URL"`
	FrontendURL             string        `mapstructure:"FRONTEND_URL"`
}

func LoadConfig(path string) (config Config, err error) {
	// Confic kck
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	viper.BindEnv("DB_SOURCE")
	viper.BindEnv("SERVER_ADDRESS")
	viper.BindEnv("TOKEN_SYMMETRIC_KEY")
	viper.BindEnv("ACCESS_TOKEN_DURATION")
	viper.BindEnv("GOOGLE_OAUTH_CLIENT_ID")
	viper.BindEnv("GOOGLE_OAUTH_CLIENT_SECRET")
	viper.BindEnv("GOOGLE_OAUTH_REDIRECT_URL")
	viper.BindEnv("FRONTEND_URL")

	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return
		}
		err = nil
	}

	err = viper.Unmarshal(&config)
	return
}
