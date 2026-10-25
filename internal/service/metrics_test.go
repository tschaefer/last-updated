/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func handleMetricsReturnsPrometheusMetrics(t *testing.T) {
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	s := Service{}
	handler := http.HandlerFunc(s.handleMetrics)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "should return OK")

	body := rr.Body.String()
	assert.Contains(t, body, "last_updated_patch_duration_seconds", "should contain defined metrics")
	assert.Contains(t, body, "last_updated_reviews_requests_total", "should contain defined metrics")
	assert.Contains(t, body, "last_updated_reviews_patched_total", "should contain defined metrics")
	assert.Contains(t, body, "last_updated_reviews_failed_total", "should contain defined metrics")
}

func TestMetrics(t *testing.T) {
	t.Run("service.handleMetrics returns prometheus metrics", handleMetricsReturnsPrometheusMetrics)
}
