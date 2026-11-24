package config

import "flag"

func ParseFlags(args []string) error {
	httpConfig := *ServiceCfg.Http
	databaseConfig := *ServiceCfg.Database
	clientConfig := *ServiceCfg.Client

	flags := flag.NewFlagSet("gophermart", flag.ContinueOnError)
	flags.Var(&httpConfig, "a", "HTTP server address (host:port)")
	flags.Var(&databaseConfig, "d", "PostgreSQL connection URI")
	flags.StringVar(&clientConfig.BaseUrl, "r", clientConfig.BaseUrl, "accrual system base URL")
	if err := flags.Parse(args); err != nil {
		return err
	}

	*ServiceCfg.Http = httpConfig
	*ServiceCfg.Database = databaseConfig
	*ServiceCfg.Client = clientConfig
	return nil
}
