package main

import (
	"oil/config"
	"oil/di"
	"oil/shared/logger"
)

func main() {
	cfg := config.Get()

	logger.InitLogger()

	logger.SetLogLevel(cfg)

	http := di.InitializeService()
	http.Serve()
}
