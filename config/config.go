package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog/log"
	"sync"
)

type Config struct {
	Server struct {
		Env      string `envconfig:"ENV"`
		LogLevel string `envconfig:"LOG_LEVEL"`
		Port     string `envconfig:"PORT"`
		Host     string `envconfig:"HOST"`
		Shutdown struct {
			CleanupPeriodSeconds int64 `envconfig:"CLEANUP_PERIOD_SECONDS"`
			GracePeriodSeconds   int64 `envconfig:"GRACE_PERIOD_SECONDS"`
		} `envconfig:"SHUTDOWN"`
	} `envconfig:"SERVER"`
}

var (
	conf        Config
	once        sync.Once
	initialized bool
)

func Init() error {
	var err error

	once.Do(func() {
		err = godotenv.Load(".env")
		if err != nil {
			log.Warn().Err(err).Msg("Could not load .env file, continuing with existing environment variables")
		} else {
			log.Info().Msg("Successfully loaded variables from .env file into environment")
		}

		err = envconfig.Process("", &conf)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to process environment variables")
		}

		initialized = true

		log.Info().Msg("Service configuration initialized successfully")
	})

	if err != nil {
		return fmt.Errorf("loading .env file: %w", err)
	}

	return nil
}

func Get() *Config {
	if !initialized {
		if err := Init(); err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize configuration")
		}
	}

	return &conf
}
