/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package service

import (
	"net/http"

	"github.com/tschaefer/k8s/webhook/last-updated/internal/version"
)

// handlePing returns info for a simple health check.
func (s *Service) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-LastUpdated-Commit", version.Commit())
	w.Header().Set("X-LastUpdated-Version", version.Release())
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
