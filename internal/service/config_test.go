/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func parseConfigReturnsErrorWhenAnnotationKeyIsMissing(t *testing.T) {
	s := Service{}
	err := s.parseConfig(&Config{})
	assert.Error(t, err, "method should fail")
	assert.ErrorContains(t, err, "annotationKey is required", "error message should match")
}

func parseConfigReturnsErrorWhenRessourcesIsMissing(t *testing.T) {
	s := Service{}
	err := s.parseConfig(&Config{AnnotationKey: "UpdatedTimestamp"})
	assert.Error(t, err, "method should fail")
	assert.ErrorContains(t, err, "at least one resource is required", "error message should match")
}

func parseConfigReturnsErrorWhenTimestampFormatIsInvalid(t *testing.T) {
	s := Service{}
	err := s.parseConfig(
		&Config{
			AnnotationKey:   "UpdatedTimestamp",
			Resources:       []Resource{{Group: "app", Resource: "deployments"}},
			TimestampFormat: "iso8601",
		},
	)
	assert.Error(t, err, "method should fail")
	assert.ErrorContains(t, err, "timestampFormat must be RFC3339 or RFC3339Nano", "error message should match")
}

func parseConfigReturnsErrorWhenResourcesResourceIsEmpty(t *testing.T) {
	s := Service{}
	err := s.parseConfig(
		&Config{
			AnnotationKey:   "UpdatedTimestamp",
			Resources:       []Resource{{Group: "app", Resource: ""}},
			TimestampFormat: "RFC3339Nano",
		},
	)
	assert.Error(t, err, "method should fail")
	assert.ErrorContains(t, err, "resource must not be empty", "error message should match")
}

func parseConfigSetsServiceFieldsWhenConfigIsValid(t *testing.T) {
	s := Service{}
	err := s.parseConfig(
		&Config{
			AnnotationKey:      "UpdatedTimestamp",
			Resources:          []Resource{{Group: "app", Resource: "deployments"}},
			TimestampFormat:    "RFC3339Nano",
			ExcludedNamespaces: []string{"kube-system"},
		},
	)
	assert.NoError(t, err, "method should succeed")
	assert.NotEqual(t, "", s.timeFormat, "field timeFormat should be set")
	assert.NotEqual(t, "", s.annotationKey, "field annotationKey should be set")
	assert.Len(t, s.resourceSet, 1, "field resourceSet should be listed")
	assert.Len(t, s.excludedNamespaces, 1, "field excludedNamespaces should be listed")
}

func TestConfig(t *testing.T) {
	t.Run("service.parseConfig returns error when annotation key is missing", parseConfigReturnsErrorWhenAnnotationKeyIsMissing)
	t.Run("service.parseConfig returns error when resources is missing", parseConfigReturnsErrorWhenRessourcesIsMissing)
	t.Run("service.parseConfig returns error when timestamp format is invalid", parseConfigReturnsErrorWhenTimestampFormatIsInvalid)
	t.Run("service.parseConfig returns error when resources resource is empty", parseConfigReturnsErrorWhenResourcesResourceIsEmpty)
	t.Run("service.parseConfig sets service fields when config is valid", parseConfigSetsServiceFieldsWhenConfigIsValid)
}
