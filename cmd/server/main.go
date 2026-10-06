package main

import (
	"context"
	"github.com/Tomassalgueiro/gorss/internal/config"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Println("Service starting...")
	log.Printf("port: %s\ndburl: %s", cfg.ServerPort, cfg.DatabaseURL)
	// main app logic
	<-ctx.Done()
	log.Println("Signal received, shutting down gracefully")

}
