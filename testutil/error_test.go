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

package testutil

import (
	"fmt"
	"testing"
)

func TestDiffErrString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		msg      string
		err      error
		wantDiff string
	}{
		{
			name: "empty_string_nil_err",
		},
		{
			name:     "empty_string_err",
			err:      fmt.Errorf("some err"),
			wantDiff: `got error "some err" but want <nil>`,
		},
		{
			name:     "non_empty_string_nil_err",
			msg:      "some err",
			wantDiff: `got error <nil> but want an error containing "some err"`,
		},
		{
			name:     "err_mismatch",
			msg:      "some err",
			err:      fmt.Errorf("other err"),
			wantDiff: `got error "other err" but want an error containing "some err"`,
		},
		{
			name: "err_match",
			msg:  "some err",
			err:  fmt.Errorf("xyz some err"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotDiff := DiffErrString(tc.err, tc.msg)
			if gotDiff != tc.wantDiff {
				t.Errorf("DiffErrString(%v, %v) got=%q, want=%q", tc.err, tc.msg, gotDiff, tc.wantDiff)
			}
		})
	}
}
