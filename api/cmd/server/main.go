package main

import (
	"os"

	"api/internal/db"
	"api/internal/httpapi"
)

func main() {
	logger := httpapi.NewLogger()
	if err := httpapi.LoadEnvFile(".env"); err != nil {
		logger.Error("dotenv", "err", err.Error())
		os.Exit(1)
	}
	cfg, err := httpapi.Load()
	if err != nil {
		logger.Error("config", "err", err.Error())
		os.Exit(1)
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		logger.Error("sqlite", "err", err.Error())
		os.Exit(1)
	}
	defer database.Close()

	srv := httpapi.New(cfg, database, logger)
	addr := ":" + cfg.Port
	logger.Info("listen", "addr", addr)
	if err := srv.Engine.Run(addr); err != nil {
		logger.Error("http", "err", err.Error())
		os.Exit(1)
	}
}
