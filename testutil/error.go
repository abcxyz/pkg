// Copyright 2022 The Authors (see AUTHORS file)
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

// Package testutil contains common util functions to facilitate tests.
package testutil

import (
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// DiffErrString returns an empty diff string if the 'got' error message contains the 'want' string.
// Otherwise returns a diff string.
func DiffErrString(got error, want string) string {
	if want == "" {
		if got == nil {
			return ""
		}
		return fmt.Sprintf("got error %q but want <nil>", got.Error())
	}
	if got == nil {
		return fmt.Sprintf("got error <nil> but want an error containing %q", want)
	}
	if msg := got.Error(); !strings.Contains(msg, want) {
		out := fmt.Sprintf("got error %q but want an error containing %q", msg, want)

		// For long strings that will be hard to visually diff, include a diff.
		// Explanation of the &&'s and ||'s: if we're diffing a long error
		// message against a short one, a detailed diff isn't needed. The
		// difference will be obvious to the eye, and any extra message will
		// just be clutter. So only show the extra diff if the messages are both
		// long, or both multi-line.
		const msgLen = 20 // chosen arbitrarily
		if len(want) >= msgLen && len(msg) >= msgLen || strings.Contains(want, "\n") && strings.Contains(msg, "\n") {
			out += fmt.Sprintf("; diff was (-got,+want):\n%s", cmp.Diff(msg, want))
		}
		return out
	}
	return ""
}
