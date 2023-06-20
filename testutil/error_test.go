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
	"strings"
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

func TestDiffErrString_LongDiff(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		msg      string
		err      error
		wantLong bool
	}{
		{
			name:     "long_errors_with_newlines",
			msg:      "this is a longish error message\nit has a few lines\nand will be hard to diff visually",
			err:      fmt.Errorf("this is ALSO a longish error message\nit has a few lines\nand will be hard to diff visually"),
			wantLong: true,
		},
		{
			name:     "short_error_with_newlines",
			msg:      "a\nb",
			err:      fmt.Errorf("a\nc"),
			wantLong: true,
		},
		{
			name:     "long_error_without_newlines",
			msg:      "one two three four five six",
			err:      fmt.Errorf("one two three four five SSSIIIIXX"),
			wantLong: true,
		},
		{
			name:     "short_error",
			msg:      "blue",
			err:      fmt.Errorf("red"),
			wantLong: false,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotDiff := DiffErrString(tc.err, tc.msg)

			// We don't try to assert the full output of cmp.Diff. It has
			// weirdness like non-breaking spaces that aren't worth the trouble
			// of writing a test for.
			gotLong := strings.Contains(gotDiff, "; diff was (-got,+want)")

			longnessStr := func(isLong bool) string {
				if isLong {
					return "long"
				}
				return "short"
			}

			if gotLong != tc.wantLong {
				t.Errorf("got a %s diff but wanted a %s diff", longnessStr(gotLong), longnessStr(tc.wantLong))
			}
		})
	}
}
