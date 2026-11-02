package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router.New(db, rdb),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// stop on Ctrl+C (SIGINT) or `docker stop` / Kubernetes (SIGTERM)
	stop, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	serverErr := make(chan error, 1)
	go func() {
		log.Println("listening on " + srv.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Fatal("server failed: ", err)
	case <-stop.Done():
		log.Println("shutdown signal received, draining requests")
	}
	cancel() // a second Ctrl+C now kills the process immediately

	// stop accepting new connections and let in-flight requests finish
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout())
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Println("forced shutdown, some requests were cut off: ", err)
	}

	// close dependencies only after no request can use them
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	log.Println("server stopped")
}

// shutdownTimeout is how long in-flight requests get to finish (SHUTDOWN_TIMEOUT, default 10s).
func shutdownTimeout() time.Duration {
	if d, err := time.ParseDuration(os.Getenv("SHUTDOWN_TIMEOUT")); err == nil && d > 0 {
		return d
	}
	return 10 * time.Second
}
