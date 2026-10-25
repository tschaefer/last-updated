/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package service

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/tschaefer/k8s/webhook/last-updated/internal/version"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// handleMutate evaluates the admission review requests and creates a
// conditional response, providing a last updated annotation patch on matching
// resources, namespaces and operations.
func (s *Service) handleMutate(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Handling mutate request", "header", r.Header)

	metricsReviewsTotal.Inc()

	if r.Method != http.MethodPost {
		metricsFailedTotal.Inc()
		http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		metricsFailedTotal.Inc()
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return
	}

	review, err := s.loadReview(r.Body)
	if err != nil {
		metricsFailedTotal.Inc()
		slog.Error("Handling mutate request", "body", r.Body, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Debug("Handling mutate request", "payload", review)

	resp, ok := s.verifyRequest(review)
	if !ok {
		slog.Debug("Handling mutate request", "verified", false)
		s.writeReview(w, review, resp)
		return
	}

	req := review.Request
	if s.excludeSubresources && req.SubResource != "" {
		slog.Debug("Handling mutate request", "subresource", req.SubResource, "skip", true)
		s.writeReview(w, review, resp)
		return
	}

	if _, found := s.excludedNamespaces[req.Namespace]; found {
		slog.Debug("Handling mutate request", "namespace", req.Namespace, "skip", true)
		s.writeReview(w, review, resp)
		return
	}

	switch req.Operation {
	case admissionv1.Update:
		// always handled
	case admissionv1.Create:
		if !s.updateOnCreate {
			s.writeReview(w, review, resp)
			return
		}
	default:
		slog.Debug("Handling mutate request", "operation", req.Operation, "skip", true)
		s.writeReview(w, review, resp)
		return
	}

	key := req.Resource.Group + "/" + req.Resource.Resource
	if _, found := s.resourceSet[key]; !found {
		slog.Debug("Handling mutate request", "resource", key, "skip", true)
		s.writeReview(w, review, resp)
		return
	}

	startTime := time.Now()
	patchOps, err := s.buildPatch(review)
	if err != nil {
		metricsFailedTotal.Inc()
		slog.Error("Handling mutate request", "patch", "failed", "error", err)
		s.writeReview(w, review, resp)
		return
	}
	metricsPatchDuration.Observe(time.Since(startTime).Seconds())

	if len(patchOps) > 0 {
		patchBytes, _ := json.Marshal(patchOps)
		pt := admissionv1.PatchTypeJSONPatch
		resp.PatchType = &pt
		resp.Patch = patchBytes

		metricsPatchedTotal.Inc()

		slog.Info("AdmissionReview patch", "uid", req.UID, "operation", req.Operation, "namespace", req.Namespace, "target", key, "subresource", (req.SubResource != ""))
	}
	slog.Debug("Handling mutate request", "response", resp)

	s.writeReview(w, review, resp)
}

func (s *Service) writeReview(w http.ResponseWriter, in admissionv1.AdmissionReview, resp *admissionv1.AdmissionResponse) {
	out := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: resp,
	}
	if in.Request != nil {
		out.Response.UID = in.Request.UID
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-LastUpdated-Commit", version.Commit())
	w.Header().Set("X-LastUpdated-Version", version.Release())
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Service) loadReview(body io.ReadCloser) (admissionv1.AdmissionReview, error) {
	var review admissionv1.AdmissionReview

	data, err := io.ReadAll(body)
	if err != nil {
		return review, err
	}

	if err := json.Unmarshal(data, &review); err != nil {
		return review, err
	}

	return review, nil
}

func (s *Service) verifyRequest(review admissionv1.AdmissionReview) (*admissionv1.AdmissionResponse, bool) {
	req := review.Request
	resp := &admissionv1.AdmissionResponse{
		Allowed: true,
	}
	if req != nil {
		resp.UID = req.UID
	}

	return resp, req != nil
}

func (s *Service) buildPatch(review admissionv1.AdmissionReview) ([]map[string]any, error) {
	req := review.Request
	var patchOps []map[string]any

	var obj map[string]any
	if err := json.Unmarshal(req.Object.Raw, &obj); err != nil {
		return patchOps, err
	}

	metadata, _ := obj["metadata"].(map[string]any)
	if metadata == nil {
		metadata = map[string]any{}
	}

	annotations, _ := metadata["annotations"].(map[string]any)
	now := time.Now().UTC().Format(s.timeFormat)

	if annotations == nil {
		patchOps = append(patchOps, map[string]any{
			"op":    "add",
			"path":  "/metadata/annotations",
			"value": map[string]string{},
		})
		annotations = map[string]any{}
	}

	cur, _ := annotations[s.annotationKey].(string)
	if cur != now {
		op := "add"
		if cur != "" {
			op = "replace"
		}
		annotationKey := s.annotationKey
		annotationKey = strings.ReplaceAll(annotationKey, "~", "~0")
		annotationKey = strings.ReplaceAll(annotationKey, "/", "~1")
		patchOps = append(patchOps, map[string]any{
			"op":    op,
			"path":  "/metadata/annotations/" + annotationKey,
			"value": now,
		})
	}

	return patchOps, nil
}
