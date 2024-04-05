package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/staterijndael/http-multiplexer/handler"
)

func main() {
	http.HandleFunc("/process", handler.ProcessHandler)
	server := &http.Server{
		Addr: ":8080",
	}

	fmt.Println("Starting server...")

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error: %v\n", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	// Graceful shutdown
	fmt.Println("Shutting down server...")
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Error during server shutdown: %v\n", err)
	}
	fmt.Println("Server stopped")
}
