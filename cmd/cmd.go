/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/goccy/go-yaml"

	"github.com/tschaefer/k8s/webhook/last-updated/internal/service"
	"github.com/tschaefer/k8s/webhook/last-updated/internal/version"
)

var (
	configPathFlag      = flag.String("config.path", "", "path to webhook configuration")
	logLevelFlag        = flag.String("log.level", "", "log level (default: info)")
	tlsCertPathFlag     = flag.String("tls.cert", "", "path to TLS certificate")
	tlsKeyPathFlag      = flag.String("tls.key", "", "path to TLS key")
	versionFlag         = flag.Bool("version", false, "show version")
	profilerEnableFlag  = flag.String("profiler.enable", "false", "enable Pyroscope push profiler")
	profilerAddressFlag = flag.String("profiler.address", "http://pyroscope:4040", "profiler server address")
)

func Execute() {
	flag.Parse()

	if *versionFlag {
		version.Print()
		os.Exit(0)
	}

	err := setLogLevel()
	checkErr(err)

	config, err := loadConfig()
	checkErr(err)

	tls, err := loadTLS()
	checkErr(err)

	profiler, err := loadProfiler()
	checkErr(err)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srvc, err := service.New(ctx, &config, &tls, &profiler)
	checkErr(err)

	err = srvc.Run()
	checkErr(err)
}

func setLogLevel() error {
	logLevel := *logLevelFlag
	if logLevel == "" {
		logLevel = os.Getenv("LOG_LEVEL")
	}
	if logLevel == "" {
		logLevel = "info"
	}

	var leveler slog.Leveler
	switch logLevel {
	case "debug":
		leveler = slog.LevelDebug
	case "info":
		leveler = slog.LevelInfo
	case "warn":
		leveler = slog.LevelWarn
	case "error":
		leveler = slog.LevelError
	default:
		return errors.New("unknown log level")
	}

	opts := &slog.HandlerOptions{
		Level: leveler,
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	return nil
}

func loadTLS() (service.TLS, error) {
	var tls service.TLS

	keyPath := *tlsKeyPathFlag
	if keyPath == "" {
		keyPath = os.Getenv("TLS_KEY_PATH")
	}
	if keyPath == "" {
		return tls, errors.New("missing TLS key path")
	}
	if _, err := os.Stat(keyPath); err != nil {
		return tls, err
	}
	tls.Key = keyPath

	certPath := *tlsCertPathFlag
	if certPath == "" {
		certPath = os.Getenv("TLS_CERTIFICATE_PATH")
	}
	if certPath == "" {
		return tls, errors.New("missing TLS certificate path")
	}
	if _, err := os.Stat(certPath); err != nil {
		return tls, err
	}
	tls.Certificate = certPath

	return tls, nil
}

func loadConfig() (service.Config, error) {
	var config service.Config

	configPath := *configPathFlag
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}
	if configPath == "" {
		return config, errors.New("missing configuration path")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return config, err
	}

	return config, nil
}

func loadProfiler() (service.Profiler, error) {
	var profiler service.Profiler

	enable := false
	enableFlag := *profilerEnableFlag
	if enableFlag == "" {
		enableFlag = os.Getenv("PROFILER_ENABLE")
	}

	if enableFlag != "" {
		pattern := regexp.MustCompile(`^(YES|yes|TRUE|true|1)$`)
		if pattern.Match([]byte(enableFlag)) {
			enable = true
		}
	}
	profiler.Enable = enable

	address := *profilerAddressFlag
	if address == "" {
		address = os.Getenv("PROFILER_ADDRESS")
	}
	profiler.Address = address

	return profiler, nil
}

func checkErr(err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
