/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
)

type AdmissionReviewPayload struct {
	Kind       string                        `json:"kind"`
	ApiVersion string                        `json:"apiVersion"`
	Response   admissionv1.AdmissionResponse `json:"response"`
}

func handleMutateReturnsMethodNotAllowedWhenMethodIsInvalid(t *testing.T) {
	req := httptest.NewRequest("GET", "/mutate", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s := Service{}
	handler := http.HandlerFunc(s.handleMutate)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func handleMutateReturnsBadRequestWhenContentTypeIsInvalid(t *testing.T) {
	req := httptest.NewRequest("POST", "/mutate", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/xml")
	rr := httptest.NewRecorder()

	s := Service{}
	handler := http.HandlerFunc(s.handleMutate)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func handleMutateReturnsBadRequestWhenJsonPayloadIsEmpty(t *testing.T) {
	req := httptest.NewRequest("POST", "/mutate", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s := Service{}
	handler := http.HandlerFunc(s.handleMutate)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func handleMutateReturnsOkWhenJsonPayloadIsInvalid(t *testing.T) {
	reqBody := map[string]string{"data": ""}
	reqPayload, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/mutate", bytes.NewReader(reqPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s := Service{}
	handler := http.HandlerFunc(s.handleMutate)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp AdmissionReviewPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	assert.Empty(t, resp.Response.UID, "response uid should not be set")
	assert.True(t, resp.Response.Allowed, "response should be allowed")
}

func handleMutateReturnsPatchWhenJsonPayloadIsValid(t *testing.T) {
	payload, err := os.OpenFile("testdata/admission_payload.json", os.O_RDONLY, 0o600)
	if err != nil {
		t.Fatalf("open payload: %v", err)
	}
	defer func() { _ = payload.Close() }()

	req := httptest.NewRequest("POST", "/mutate", payload)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	resourceSet := make(map[string]bool, 1)
	resourceSet["/configmaps"] = true
	s := Service{
		annotationKey: "example.com/lat-updated",
		resourceSet:   resourceSet,
		timeFormat:    time.RFC3339,
	}
	handler := http.HandlerFunc(s.handleMutate)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp AdmissionReviewPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	assert.NotEmpty(t, resp.Response.UID, "response uid should be set")
	assert.True(t, resp.Response.Allowed, "response should be allowed")
	assert.NotEmpty(t, resp.Response.PatchType, "response patch type should be set")
	assert.NotEmpty(t, resp.Response.Patch, "response should be available")
}

func handleMutateReturnsOkWhenJsonPayloadContainsSkippedNamespace(t *testing.T) {
	payload, err := os.OpenFile("testdata/admission_payload.json", os.O_RDONLY, 0o600)
	if err != nil {
		t.Fatalf("open payload: %v", err)
	}
	defer func() { _ = payload.Close() }()

	req := httptest.NewRequest("POST", "/mutate", payload)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	resourceSet := make(map[string]bool, 1)
	resourceSet["/configmaps"] = true
	excludedNamespaces := make(map[string]bool, 1)
	excludedNamespaces["default"] = true
	s := Service{
		annotationKey:      "example.com/last-updated",
		excludedNamespaces: excludedNamespaces,
		resourceSet:        resourceSet,
		timeFormat:         time.RFC3339,
	}
	handler := http.HandlerFunc(s.handleMutate)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp AdmissionReviewPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	assert.NotEmpty(t, resp.Response.UID, "response uid should be set")
	assert.True(t, resp.Response.Allowed, "response should be allowed")
	assert.Empty(t, resp.Response.PatchType, "response patch type should not be set")
	assert.Empty(t, resp.Response.Patch, "response should not be available")
}

func handleMutateReturnsOkWhenJsonPayloadContainsSkippedSubresource(t *testing.T) {
	payload, err := os.OpenFile("testdata/admission_subresource_payload.json", os.O_RDONLY, 0o600)
	if err != nil {
		t.Fatalf("open payload: %v", err)
	}
	defer func() { _ = payload.Close() }()

	req := httptest.NewRequest("POST", "/mutate", payload)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s := Service{
		annotationKey:       "example.com/last-updated",
		excludeSubresources: true,
	}
	handler := http.HandlerFunc(s.handleMutate)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp AdmissionReviewPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	assert.NotEmpty(t, resp.Response.UID, "response uid should be set")
	assert.True(t, resp.Response.Allowed, "response should be allowed")
	assert.Empty(t, resp.Response.PatchType, "response patch type should not be set")
	assert.Empty(t, resp.Response.Patch, "response should not be available")
}

func handleMutateReturnsOkWhenJsonPayloadContainsUndefinedTarget(t *testing.T) {
	payload, err := os.OpenFile("testdata/admission_payload.json", os.O_RDONLY, 0o600)
	if err != nil {
		t.Fatalf("open payload: %v", err)
	}
	defer func() { _ = payload.Close() }()

	req := httptest.NewRequest("POST", "/mutate", payload)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	resourceSet := make(map[string]bool, 1)
	resourceSet["apps/deployments"] = true
	s := Service{
		annotationKey: "example.com/last-updated",
		resourceSet:   resourceSet,
		timeFormat:    time.RFC3339,
	}
	handler := http.HandlerFunc(s.handleMutate)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp AdmissionReviewPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	assert.NotEmpty(t, resp.Response.UID, "response uid should be set")
	assert.True(t, resp.Response.Allowed, "response should be allowed")
	assert.Empty(t, resp.Response.PatchType, "response patch type should not be set")
	assert.Empty(t, resp.Response.Patch, "response should not be available")
}

func handleMutateReturnsPatchWhenJsonPayloadHasNoAnnotations(t *testing.T) {
	payload, err := os.OpenFile("testdata/admission_no_annotations_payload.json", os.O_RDONLY, 0o600)
	if err != nil {
		t.Fatalf("open payload: %v", err)
	}
	defer func() { _ = payload.Close() }()

	req := httptest.NewRequest("POST", "/mutate", payload)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	resourceSet := make(map[string]bool, 1)
	resourceSet["/configmaps"] = true
	s := Service{
		annotationKey: "example.com/last-updated",
		resourceSet:   resourceSet,
		timeFormat:    time.RFC3339,
	}
	handler := http.HandlerFunc(s.handleMutate)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp AdmissionReviewPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	assert.NotEmpty(t, resp.Response.UID, "response uid should be set")
	assert.True(t, resp.Response.Allowed, "response should be allowed")
	assert.NotEmpty(t, resp.Response.PatchType, "response patch type should be set")
	assert.NotEmpty(t, resp.Response.Patch, "response should be available")
}

func TestMutate(t *testing.T) {
	t.Run("service.handleMutate returns method not allowed when method is invalid", handleMutateReturnsMethodNotAllowedWhenMethodIsInvalid)
	t.Run("service.handleMutate returns bad request when content type is invalid", handleMutateReturnsBadRequestWhenContentTypeIsInvalid)
	t.Run("service.handleMutate returns bad request when JSON payload is empty", handleMutateReturnsBadRequestWhenJsonPayloadIsEmpty)
	t.Run("service.handleMutate returns ok when JSON payload is invalid", handleMutateReturnsOkWhenJsonPayloadIsInvalid)
	t.Run("service.handleMutate returns patch when JSON payload is valid", handleMutateReturnsPatchWhenJsonPayloadIsValid)
	t.Run("service.handleMutate returns ok when JSON payload contains skipped namespace", handleMutateReturnsOkWhenJsonPayloadContainsSkippedNamespace)
	t.Run("service.handleMutate returns ok when JSON payload contains skipped subresource", handleMutateReturnsOkWhenJsonPayloadContainsSkippedSubresource)
	t.Run("service.handleMutate returns ok when JSON payload contains undefinded target", handleMutateReturnsOkWhenJsonPayloadContainsUndefinedTarget)
	t.Run("service.handleMutate returns ok when JSON payload has no annotations", handleMutateReturnsPatchWhenJsonPayloadHasNoAnnotations)
}
