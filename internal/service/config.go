/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package service

import (
	"errors"
	"log/slog"
	"strings"
	"time"
)

// parseConfig evaluates the admission webhook config and provides the
// required information within the service object.
func (s *Service) parseConfig(config *Config) error {
	slog.Debug("Parsing configuration", "config", config)

	if config.AnnotationKey == "" {
		return errors.New("annotationKey is required")
	}
	s.annotationKey = config.AnnotationKey

	if len(config.Resources) == 0 {
		return errors.New("at least one resource is required")
	}

	var timeFormat string
	switch config.TimestampFormat {
	case "", "RFC3339":
		timeFormat = time.RFC3339
	case "RFC3339Nano":
		timeFormat = time.RFC3339Nano
	default:
		return errors.New("timestampFormat must be RFC3339 or RFC3339Nano")
	}
	s.timeFormat = timeFormat

	resourceSet := make(map[string]bool, len(config.Resources))
	for _, t := range config.Resources {
		g := t.Group
		r := t.Resource
		if r == "" {
			return errors.New("resource must not be empty")
		}
		resourceSet[g+"/"+r] = true
	}
	s.resourceSet = resourceSet

	excludedNamespaces := make(map[string]bool, len(config.ExcludedNamespaces))
	for _, ns := range config.ExcludedNamespaces {
		ns = strings.TrimSpace(ns)
		if ns != "" {
			excludedNamespaces[ns] = true
		}
	}
	s.excludedNamespaces = excludedNamespaces

	s.excludeSubresources = config.ExcludeSubresources
	s.updateOnCreate = config.UpdateOnCreate

	return nil
}
