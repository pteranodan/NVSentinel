package version

import (
	"fmt"
	"runtime"
	"testing"
)

func TestUserAgent(t *testing.T) {
	originalVersion := GitVersion
	defer func() { GitVersion = originalVersion }()

	GitVersion = "v1.2.3-test"

	expected := fmt.Sprintf("nvidia-device-api-client/v1.2.3-test (%s/%s)", runtime.GOOS, runtime.GOARCH)
	if got := UserAgent(); got != expected {
		t.Errorf("UserAgent() = %q, want %q", got, expected)
	}
}
