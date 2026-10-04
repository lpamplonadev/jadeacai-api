package main

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/lpamplonadev/jadeacai-bkend/internal/application"
	"github.com/lpamplonadev/jadeacai-bkend/internal/infrastructure/postgres"
	"github.com/lpamplonadev/jadeacai-bkend/internal/transport/httpapi"
)

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load .env: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store, err := postgres.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer store.Close()

	service := application.NewService(store)
	if err := httpapi.NewRouter(service, os.Getenv("ADMIN_API_KEY")).Run(":" + port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
