/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package version

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func __capture(f func()) string {
	originalStdout := os.Stdout

	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = originalStdout

	var buf = make([]byte, 5096)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func releaseReturnsVersion(t *testing.T) {
	expected := "1.0.0"
	Version = expected

	assert.Equal(t, expected, Release(), "version should match expected")
}

func releaseReturnsStringDevWhenVersionIsEmpty(t *testing.T) {
	Version = ""

	assert.Equal(t, "dev", Release(), "version should be 'dev' when empty")
}

func commitReturnsCommitHash(t *testing.T) {
	expected := "abc123def"
	GitCommit = expected

	assert.Equal(t, expected, Commit(), "commit hash should match expected")
}

func bannerReturnsString(t *testing.T) {
	assert.IsType(t, "", Banner(), "banner should be a string")
	assert.Len(t, Banner(), 336, "banner should not be empty")
}

func printOutputsBannerAndInfo(t *testing.T) {
	Version = "1.0.0"
	GitCommit = "abc123def"

	output := __capture(Print)

	assert.Contains(t, output, Banner(), "output should contain banner")
	assert.Contains(t, output, "Release: 1.0.0", "output should contain release info")
	assert.Contains(t, output, "Commit:  abc123def", "output should contain commit info")
	assert.Contains(t, output, "\033[34m", "output should contain color start escape sequence")
	assert.Contains(t, output, "\033[0m", "output should contain color stop escape sequence")
}

func printOutputsBannerAndInfoWithNoColorWhenEnvVarIsSet(t *testing.T) {
	Version = "1.0.0"
	GitCommit = "abc123def"

	_ = os.Setenv("NO_COLOR", "1")
	defer func() {
		_ = os.Unsetenv("NO_COLOR")
	}()

	output := __capture(Print)

	assert.Contains(t, output, Banner(), "output should contain banner")
	assert.Contains(t, output, "Release: 1.0.0", "output should contain release info")
	assert.Contains(t, output, "Commit:  abc123def", "output should contain commit info")
	assert.NotContains(t, output, "\033[34m", "output should not contain color start escape sequence")
	assert.NotContains(t, output, "\033[0m", "output should not contain color stop escape sequence")
}

func TestVersion(t *testing.T) {
	t.Run("version.Release returns version", releaseReturnsVersion)
	t.Run("version.Release returns string dev when version is empty", releaseReturnsStringDevWhenVersionIsEmpty)
	t.Run("version.Commit returns commit hash", commitReturnsCommitHash)
	t.Run("version.Banner returns string", bannerReturnsString)
	t.Run("version.Print outputs banner and info", printOutputsBannerAndInfo)
	t.Run("version.Print outputs banner and info with no color when env var is set", printOutputsBannerAndInfoWithNoColorWhenEnvVarIsSet)
}
