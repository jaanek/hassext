package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	flag "github.com/spf13/pflag"
	"github.com/zerodha/logf"
)

func initLogger(ko *koanf.Koanf) logf.Logger {
	opts := logf.Opts{EnableCaller: false}
	if ko.Bool("app.debug") {
		opts.Level = logf.DebugLevel
		opts.EnableColor = true
	}
	return logf.New(opts)
}

func initConfig() (*koanf.Koanf, error) {
	var (
		ko = koanf.New(".")
		f  = flag.NewFlagSet("front", flag.ContinueOnError)
	)

	// Configure flags
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}
	// Register "--config" flag
	cfgPath := f.String("config", "config.sample.toml", "Path to a config file to load")

	// Parse and load flags
	err := f.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	// Load configuration
	err = ko.Load(file.Provider(*cfgPath), toml.Parser())
	if err != nil {
		return nil, err
	}
	err = ko.Load(env.Provider("HASSEXT_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "HASSEXT_")), "__", ".", -1)
	}), nil)
	if err != nil {
		return nil, err
	}

	return ko, nil
}
