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
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to initialize db, %v", err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Println("Service starting...")
	log.Printf("port: %s\ndburl: %s", cfg.ServerPort, cfg.DatabaseURL)

	// main app logic
	<-ctx.Done()
	log.Println("Signal received, shutting down gracefully")
}
