// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"testing"
)

//nolint:paralleltest // Can't be paralleled because of t.Setenv
func TestIsIntegration(t *testing.T) {
	if IsIntegration(t) {
		t.Errorf("IsIntegration() got 'true' want 'false'")
	}

	t.Setenv("TEST_INTEGRATION", "true")
	if !IsIntegration(t) {
		t.Errorf("IsIntegration() got 'false' want 'true'")
	}
}

//nolint:paralleltest // Can't be paralleled because of t.Setenv
func TestIsIntegrationMain(t *testing.T) {
	if IsIntegrationMain() {
		t.Errorf("IsIntegrationMain() got 'true' want 'false'")
	}

	t.Setenv("TEST_INTEGRATION", "true")
	if !IsIntegrationMain() {
		t.Errorf("IsIntegrationMain() got 'false' want 'true'")
	}
}
