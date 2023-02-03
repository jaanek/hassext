package main

import (
	"fmt"
	"os"

	hass "github.com/jaanek/hassext"
	"github.com/jaanek/hassext/cmd"
	"github.com/jaanek/hassext/emodul"
	"github.com/zerodha/logf"
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

	// Initialize hassext
	h, err := hass.Init(ko, lo)
	if err != nil {
		lo.Error("hassext initialization failed", "error", err)
		os.Exit(-1)
	}

	// run command
	err = run(lo, h)
	if err != nil {
		lo.Error("command failed", "error", err)
		os.Exit(-1)
	}
}

func run(lo logf.Logger, h *hass.HassExt) error {
	// Init emodul
	if err := h.Emodul.Init(); err != nil {
		lo.Error("eModul init", "failed", err)
		return err
	}

	// set working mode
	_, err := h.Emodul.SetWorkingMode(emodul.HOUSE_HEATING)
	if err != nil {
		lo.Error("Error while setting working mode", "to", emodul.HOUSE_HEATING, "err", err)
		return err
	}
	return nil
}
