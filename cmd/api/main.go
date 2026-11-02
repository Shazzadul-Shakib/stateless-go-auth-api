package main

import (
	"context"
	"log"
	"os"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/modules/auth"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/platform"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/router"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := platform.NewPostgres(ctx, dbURL)
	if err != nil {
		log.Fatal("failed to connect to postgres: ", err)
	}
	log.Println("connected to postgres")

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL is not set")
	}

	rdb, err := platform.NewRedis(ctx, redisURL)
	if err != nil {
		log.Fatal("failed to connect to redis: ", err)
	}
	defer rdb.Close()
	log.Println("connected to redis")

	if err := db.AutoMigrate(&auth.User{}); err != nil {
		log.Fatal("failed to migrate database: ", err)
	}
	log.Println("database migrated")

	router := router.New(db)
	if err := router.Run(); err != nil {
		log.Fatal("server failed: ", err)
	}
}
