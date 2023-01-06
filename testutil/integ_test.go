package testutil

import (
	"testing"
)

func TestIsIntegration(t *testing.T) {
	t.Parallel()

	if IsIntegration() {
		t.Errorf("IsIntegration() got 'true' want 'false'")
	}

	t.Setenv("TEST_INTEGRATION", "true")
	if !IsIntegration() {
		t.Errorf("IsIntegration() got 'false' want 'true'")
	}
}
