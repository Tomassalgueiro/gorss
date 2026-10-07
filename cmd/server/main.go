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

	"github.com/Tomassalgueiro/gorss/internal/config"
	"github.com/Tomassalgueiro/gorss/internal/database"
	"github.com/Tomassalgueiro/gorss/internal/feed"
	"github.com/Tomassalgueiro/gorss/internal/server"
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

	repo := feed.NewRepository(db)
	h := server.NewHandler(repo)
	mux := h.Routes()

	log.Println("Service starting...")
	log.Printf("listening on port: %s\ndburl: %s", cfg.ServerPort, cfg.DatabaseURL)

	srv := &http.Server{
		Addr: ":" + cfg.ServerPort, 
		Handler: mux,
		ReadTimeout: 5 * time.Second, 
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 120 * time.Second,
	}
	
	go func() {
	    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
    	    }
	}()

	<-ctx.Done()
	log.Println("Shutting down HTTP server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		log.Fatalf("Server forced to Shutdown: %v", err)
	}
	log.Println("Signal received, shutting down gracefully")
}
