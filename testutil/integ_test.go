package testutil

import (
	"testing"
)

func TestIsIntegration(t *testing.T) {
	// Can't be paralleled since we set env var.

	if IsIntegration(t) {
		t.Errorf("IsIntegration() got 'true' want 'false'")
	}

	t.Setenv("TEST_INTEGRATION", "true")
	if !IsIntegration(t) {
		t.Errorf("IsIntegration() got 'false' want 'true'")
	}
}
