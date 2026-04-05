package config

import (
	"log"
	"os"
)

type Config struct {
	GroqAPIKey string
	DBPath          string
	Port            string
}

func Load() *Config {
	key := os.Getenv("GROQ_API_KEY")
	if key == "" {
		log.Fatal("GROQ_API_KEY 环境变量未设置")
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "ewords.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return &Config{
		GroqAPIKey: key,
		DBPath:          dbPath,
		Port:            port,
	}
}
