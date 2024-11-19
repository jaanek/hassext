package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	hass "github.com/jaanek/hassext"
	"github.com/jaanek/hassext/cmd"
)

var (
	buildString = "unknown"
)

func main() {
	// Initialize and load the config
	ko, err := cmd.InitConfig()
	if err != nil {
		fmt.Printf("error loading config %v", err)
		os.Exit(-1)
	}
	var lo = cmd.InitLogger(ko)
	lo.Info("booting hassext server", "version", buildString)

	// Initialize hassext
	h, err := hass.Init(ko, lo)
	if err != nil {
		lo.Error("hassext initialization failed", "error", err)
		os.Exit(-1)
	}

	// Create a channel to listen cancellation signals
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// Run the home assistant extension
	err = h.Run(ctx)
	if err != nil {
		lo.Error("run error", "error", err)
		cancel()
	}

	// Listen on a close channel until interrupt is received
	<-ctx.Done()

	// Cancel the context to gracefully shutdown and perform app cleanup
	cancel()
	h.Shutdown()
}
