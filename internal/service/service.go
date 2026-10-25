/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/grafana/pyroscope-go"

	"github.com/tschaefer/k8s/webhook/last-updated/internal/version"
)

// Resource identifies a resource by its group and pluralized name (kind)
type Resource struct {
	// Group describes the resource related group.
	Group string `json:"group"`

	// Resource describes the resource related pluralized name (kind).
	Resource string `json:"resource"`
}

// Config holds the admission review conditions.
type Config struct {
	// AnnotationKey names the last updated annotation.
	AnnotationKey string `json:"annotationKey"`

	// ExcludeSubresources defines, if subresources updates trigger timestamp patch or not.
	ExcludeSubresources bool `json:"excludeSubresources"`

	// ExcludesNamespaces provides a list of namespaces to skip.
	ExcludedNamespaces []string `json:"excludedNamespaces"`

	// Resources provides a list of resources to review.
	Resources []Resource `json:"resources"`

	// TimestampFormat defines the timestamp accuracy. Valid strings are RFC3339 or RFC3339Nano.
	TimestampFormat string `json:"timestampFormat"`

	// UpdateOnCreate defines, if the last update annotation should be patched or not on resource creation.
	UpdateOnCreate bool `json:"updateOnCreate"`
}

// TLS provides the paths of a cert and key pair.
type TLS struct {
	// Certificate describes the path to a cert in PEM format.
	Certificate string `json:"certificate"`

	// Key describes the path to a key in PEM format.
	Key string `json:"key"`
}

// Profiler configuration.
type Profiler struct {
	// Enable profiling
	Enable bool `json:"enable"`

	// Pyroscope server address
	Address string `json:"address"`
}

// Service keeps the service object attributes.
type Service struct {
	ctx                 context.Context
	config              *Config
	tls                 *TLS
	pyroscope           *pyroscope.Profiler
	annotationKey       string
	excludeSubresources bool
	excludedNamespaces  map[string]bool
	resourceSet         map[string]bool
	timeFormat          string
	updateOnCreate      bool
}

const (
	// WebhookListenAddress provides the listen address of the admission webhook service.
	WebhookListenAddress = ":8443"

	// MetricsListenAddress provides the listen address of the metrics service.
	MetricsListenAddress = ":9090"

	// HealthListenAddress provides the listen address of the health service.
	HealthListenAddress = ":8080"
)

// New creates an unstarted service instance.
func New(ctx context.Context, config *Config, tls *TLS, profiler *Profiler) (*Service, error) {
	slog.Debug("Initializing service")

	if reflect.DeepEqual(*config, Config{}) {
		return nil, errors.New("config must not be empty")
	}

	if reflect.DeepEqual(*tls, TLS{}) {
		return nil, errors.New("tls must not be empty")
	}

	var pyro *pyroscope.Profiler
	if profiler.Enable {
		slog.Debug("Start profiler", "cfg", profiler)
		var err error
		pyro, err = pyroscope.Start(pyroscope.Config{
			ApplicationName: "last-updated-webhook",
			ServerAddress:   profiler.Address,
			Logger:          nil,
			ProfileTypes: []pyroscope.ProfileType{
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,
				pyroscope.ProfileGoroutines,
				pyroscope.ProfileMutexCount,
				pyroscope.ProfileMutexDuration,
				pyroscope.ProfileBlockCount,
				pyroscope.ProfileBlockDuration,
			},
		})
		if err != nil {
			return nil, err
		}
	}

	s := &Service{
		ctx:       ctx,
		config:    config,
		tls:       tls,
		pyroscope: pyro,
	}
	if err := s.parseConfig(config); err != nil {
		return nil, err
	}

	return s, nil
}

// Run starts service listening on WebhookListenAddress and metrics service
// listening on MetricsListenAddress.
func (s *Service) Run() error {
	slog.Debug("Running service", "tls", s.tls)

	slog.Info("last-updated AdmissionWebhook", "commit", version.Commit(), "release", version.Release())

	srv := s.createWebhookService()
	msrv := s.createMetricsService()
	hsrv := s.createHealthService()

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	errCh := make(chan error, 1)

	s.runWebhookService(srv, errCh)
	s.runMetricsService(msrv, errCh)
	s.runHealthService(hsrv, errCh)

	stopServices := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s.stopWebhookService(srv, ctx)
		s.stopMetricsService(msrv, ctx)
		s.stopHealthService(hsrv, ctx)
	}

	select {
	case <-ctx.Done():
		stopServices()
	case err := <-errCh:
		stopServices()

		if err != nil {
			return err
		}
	}

	close(errCh)

	if s.pyroscope != nil {
		slog.Debug("Stop profiler")
		_ = s.pyroscope.Stop()
	}

	return nil
}

// Create webhook service and register handlers.
func (s *Service) createWebhookService() *http.Server {
	slog.Info("Creating AdmissionWebhook service")
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", s.handleMutate)

	return &http.Server{
		Addr:         WebhookListenAddress,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Run webhook service in background.
func (s *Service) runWebhookService(srv *http.Server, errCh chan (error)) {
	slog.Info("Starting AdmissionWebhook service", "listen", WebhookListenAddress)
	go func() {
		err := srv.ListenAndServeTLS(s.tls.Certificate, s.tls.Key)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			err = fmt.Errorf("AdmissionWebhook service: %v", err) // nolint:staticcheck // (ST1005)
			errCh <- err
			return
		}
	}()
}

// Stop webhook service.
func (s *Service) stopWebhookService(srv *http.Server, ctx context.Context) {
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("AdmissionWebhook service shutdown", "error", err)
		return
	}

	slog.Info("Stopped AdmissionWebhook service")
}

// Create metrics service and register handlers.
func (s *Service) createMetricsService() *http.Server {
	slog.Info("Creating Metrics service")
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", s.handleMetrics)

	return &http.Server{
		Addr:         MetricsListenAddress,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Run metrics service in background.
func (s *Service) runMetricsService(srv *http.Server, errCh chan (error)) {
	slog.Info("Starting Metrics service", "listen", MetricsListenAddress)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			err = fmt.Errorf("Metrics service: %v", err) // nolint:staticcheck // (ST1005)
			errCh <- err
			return
		}
	}()
}

// Stop metrics service.
func (s *Service) stopMetricsService(srv *http.Server, ctx context.Context) {
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Metrics service shutdown", "error", err)
		return
	}

	slog.Info("Stopped Metrics service")
}

// Create health service and register handlers.
func (s *Service) createHealthService() *http.Server {
	slog.Info("Creating Health service")
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", s.handlePing)

	return &http.Server{
		Addr:         HealthListenAddress,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Run health service in background.
func (s *Service) runHealthService(srv *http.Server, errCh chan (error)) {
	slog.Info("Starting Health service", "listen", HealthListenAddress)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			err = fmt.Errorf("Health service: %v", err) // nolint:staticcheck // (ST1005)
			errCh <- err
			return
		}
	}()
}

// Stop health service.
func (s *Service) stopHealthService(srv *http.Server, ctx context.Context) {
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Health service shutdown", "error", err)
		return
	}

	slog.Info("Stopped Health service")
}
