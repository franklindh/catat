package util

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	DBDriver      string `mapstructure:"DB_DRIVER"`
	DBSource      string `mapstructure:"DB_SOURCE"`
	ServerAddress string `mapstructure:"SERVER_ADDRESS"`
}

func LoadConfig(path string) (config Config, err error) {
	envServerAddr := os.Getenv("SERVER_ADDRESS")
	envDBSource := os.Getenv("DB_SOURCE")
	envDBDriver := os.Getenv("DB_DRIVER")

	if envServerAddr != "" {
		config.ServerAddress = envServerAddr
	}
	if envDBSource != "" {
		config.DBSource = envDBSource
	}
	if envDBDriver != "" {
		config.DBDriver = envDBDriver
	}

	if config.ServerAddress == "" || config.DBSource == "" {
		viper.AddConfigPath(path)
		viper.SetConfigName("app")
		viper.SetConfigType("env")

		viper.AutomaticEnv()

		err = viper.ReadInConfig()
		if err != nil {
			log.Printf("Warning: Cannot read config file: %v", err)
		} else {
			viper.Unmarshal(&config)
		}
	}

	if config.ServerAddress == "" {
		config.ServerAddress = ":3000"
	}
	if config.DBSource == "" {
		config.DBSource = "postgresql://postgres:password@localhost:5432/catat_db?sslmode=disable"
	}
	if config.DBDriver == "" {
		config.DBDriver = "postgres"
	}

	return
}
