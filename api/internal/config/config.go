package config

import "os"

type Config struct {
	PostgresURL string
	KafkaBroker string
	GRPCPort    string
}

func Load() *Config {
	return &Config{
		PostgresURL: os.Getenv("POSTGRES_URL"),
		KafkaBroker: os.Getenv("KAFKA_BROKER"),
		GRPCPort:    os.Getenv("GRPC_PORT"),
	}
}
