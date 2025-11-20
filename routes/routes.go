package routes

import (
	"fmt"
	"net/http"

	"github.com/b-j-roberts/foc-engine/internal/config"
	routeutils "github.com/b-j-roberts/foc-engine/routes/utils"
)

func InitBaseRoutes() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		routeutils.SetupHeaders(w)
		w.WriteHeader(http.StatusOK)
	})
}

func InitRoutes() {
	InitBaseRoutes()
	if config.ModuleEnabled(config.ModuleRegistry) {
		InitRegistryRoutes()
	}
	if config.ModuleEnabled(config.ModuleAccounts) {
		InitAccountsRoutes()
	}
	if config.ModuleEnabled(config.ModuleEvents) {
		InitEventsRoutes()
	}
	if config.ModuleEnabled(config.ModulePaymaster) {
		InitPaymasterRoutes()
	}
	// Device verification routes are always available if service is initialized
	InitDeviceVerificationRoutes()
}

func StartServer(host string, port int) {
	InitRoutes()
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting server on %s\n", addr)
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Println("Error starting server:", err)
		}
		fmt.Println("Server stopped")
	}()
}
