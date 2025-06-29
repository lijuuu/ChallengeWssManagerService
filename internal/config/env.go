package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ChallengeGRPCPort string
	ChallengeHTTPPort string
	PsqlURL           string
	SessionSecretKey  string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}
	config := Config{
		ChallengeGRPCPort: getEnv("CHALLENGEGRPCPORT", "50057"),
		PsqlURL:           getEnv("PSQLURL", "host=localhost port=5432 user=admin password=password dbname=xcodedev sslmode=disable"),
		ChallengeHTTPPort: getEnv("CHALLENGEHTTPPORT", "3333"),
		SessionSecretKey:  getEnv("SESSIONSECRETKEY", "something"),
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
