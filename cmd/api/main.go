package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/b-j-roberts/foc-engine/internal/config"
	"github.com/b-j-roberts/foc-engine/internal/db/mongo"
	"github.com/b-j-roberts/foc-engine/internal/deviceverification"
	"github.com/b-j-roberts/foc-engine/routes"
)

func main() {
	config.InitConfig()

	if mongo.ShouldConnectMongo() {
		mongo.InitMongoDB()
	}

	// Initialize device verification service
	if err := deviceverification.InitDeviceVerification(); err != nil {
		fmt.Printf("Warning: Failed to initialize device verification: %v\n", err)
		fmt.Println("Device verification endpoints will not be available")
	}

	routes.StartServer(config.Conf.Api.Host, config.Conf.Api.Port)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})
	for {
		select {
		case <-done:
			fmt.Println("Connection closed")
			return
		case <-interrupt:
			fmt.Println("Interrupt signal received, shutting down...")
			close(done)
			return
		default:
			// Do nothing, just keep the connection alive
		}
	}
}
