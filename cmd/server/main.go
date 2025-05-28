// cmd/server/main.go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"reddit-orchestrator/internal/app"
)

func main() {
	application, err := app.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}


	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v. Shutting down...", sig)
		application.Shutdown()
		os.Exit(0)
	}()

	log.Println("Starting Reddit Subreddit Orchestrator...")
	log.Println("BlueBerry dashboard available at http://localhost:8080")
	log.Println("Login with configured username/password")

	// Start the scheduler and API server
	if err := application.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
}