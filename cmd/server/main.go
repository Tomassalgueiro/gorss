package main

import (
	"context"
	"github.com/Tomassalgueiro/gorss/internal/config"
	"github.com/Tomassalgueiro/gorss/internal/database"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to initialize db, %v", err)
	}
	err = database.Migrate(db)
	if err != nil {
		log.Fatalf("failed to migrate db, %v", err)
	}
	defer db.Close()

	log.Println("Service starting...")
	log.Printf("listening on port: %s\ndburl: %s", cfg.ServerPort, cfg.DatabaseURL)

	<-ctx.Done()
	log.Println("Signal received, shutting down gracefully")
}
