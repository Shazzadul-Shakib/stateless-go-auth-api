package main

import (
	"context"
	"log"
	"os"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/platform"
	"github.com/gin-gonic/gin"
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

	pool, err := platform.NewPostgres(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	log.Println("connected to postgres")

	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	router.Run()
}
