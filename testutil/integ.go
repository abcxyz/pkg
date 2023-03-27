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
	"log"
	"os"
	"strconv"
	"testing"
)

// IsIntegration checks env var TEST_INTEGRATION and consider that we're in an
// integration test if it's set to true.
func IsIntegration(tb testing.TB) bool {
	tb.Helper()
	integVal := os.Getenv("TEST_INTEGRATION")
	if integVal == "" {
		return false
	}
	isInteg, err := strconv.ParseBool(integVal)
	if err != nil {
		tb.Fatalf("failed to parse TEST_INTEGRATION: %v", err)
	}
	return isInteg
}

// SkipIfNotIntegration skips the test if [IsIntegration] returns false.
func SkipIfNotIntegration(tb testing.TB) {
	tb.Helper()
	if !IsIntegration(tb) {
		tb.Skip("Not integration test, skipping")
	}
}

// IsIntegrationMain checks env var TEST_INTEGRATION and consider that we're in
// an integration test if it's set to true. This func should be used in
// [TestMain] where [testing.TB] is not accessible. Otherwise, use
// [IsIntegration] instead.
//
// [TestMain]: https://pkg.go.dev/testing#hdr-Main
func IsIntegrationMain() bool {
	integVal := os.Getenv("TEST_INTEGRATION")
	if integVal == "" {
		return false
	}
	isInteg, err := strconv.ParseBool(integVal)
	if err != nil {
		log.Fatalf("failed to parse TEST_INTEGRATION: %v", err)
	}
	return isInteg
}
