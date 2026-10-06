package config

import "os"

type Config struct {
	ServerPort string
	DatabaseURL string
}

// default port is 6060
// default db url is gorss.db
func New() *Config{

	ServerPort := "6060"
	DatabaseURL := "gorss.db"
	c := Config{ServerPort: ServerPort, DatabaseURL: DatabaseURL}	

	if val := os.Getenv("PORT"); val != ""{
		c.ServerPort = val
	}

	if val := os.Getenv("DB_URL"); val != ""{
		c.DatabaseURL = val
	}

	return &c

}
