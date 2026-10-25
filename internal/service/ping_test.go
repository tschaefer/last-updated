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

func handlePingReturnsHttpCodeOK(t *testing.T) {
	req := httptest.NewRequest("GET", "/ping", nil)
	rr := httptest.NewRecorder()

	s := Service{}
	handler := http.HandlerFunc(s.handlePing)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "should return OK")
}

func TestPing(t *testing.T) {
	t.Run("service.handlePing returns http code OK", handlePingReturnsHttpCodeOK)
}
