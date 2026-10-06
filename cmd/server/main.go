package main

import (
	"log"
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main(){

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Println("Server started with success")
	// main app logic 
	<-ctx.Done()
	log.Println("Signal received, shuting down gracefully")

}
