package httpapi

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnvFile reads path into the process environment without overriding
// variables already set. A missing file is ignored.
func LoadEnvFile(path string) error {
	if path == "" {
		path = ".env"
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return godotenv.Load(path)
}

type Config struct {
	Salt        string
	DBPath      string
	Port        string
	CORSOrigins []string
}

func Load() (Config, error) {
	salt := os.Getenv("LINK_ID_SALT")
	if salt == "" {
		return Config{}, fmt.Errorf("LINK_ID_SALT is required")
	}
	dbPath := os.Getenv("LINKS_DB_PATH")
	if dbPath == "" {
		dbPath = "data/links.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{
		Salt:        salt,
		DBPath:      dbPath,
		Port:        port,
		CORSOrigins: []string{"http://localhost:4200", "http://127.0.0.1:4200"},
	}, nil
}
