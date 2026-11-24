package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

func ParseEnv() error {
	values := struct {
		RunAddress           string `env:"RUN_ADDRESS"`
		DatabaseURI          string `env:"DATABASE_URI"`
		AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	}{}
	if err := env.Parse(&values); err != nil {
		return err
	}

	if values.DatabaseURI != "" {
		ServiceCfg.Database.Dsn = values.DatabaseURI
	}
	if values.AccrualSystemAddress != "" {
		ServiceCfg.Client.BaseURL = values.AccrualSystemAddress
	}
	if values.RunAddress != "" {
		if err := ServiceCfg.HTTP.Set(values.RunAddress); err != nil {
			return fmt.Errorf("RUN_ADDRESS: %w", err)
		}
	}
	return nil
}
