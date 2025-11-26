// Copyright 2025 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package run

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/testutil"
)

// This may not be portable due to depending on behavior of external programs.
func TestSimple(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		args       []string
		wantOut    string
		wantStdErr string
		wantCode   int
		wantErr    string
	}{
		{
			name:    "happy_path",
			args:    []string{"echo", "foo", "bar   bazz"},
			wantOut: "foo bar   bazz\n",
		},
		{
			name:    "no_command_error",
			args:    []string{"echoooocrapimistyped", "foo", "bar   bazz"},
			wantErr: "not found",
		},
		{
			name:       "exit_code_error",
			args:       []string{"cat", "/fake/file/path/should/fail"},
			wantStdErr: "cat: /fake/file/path/should/fail: No such file or directory\n",
			wantErr:    "exited non-zero (1): exit status 1 (context error: <nil>)\nstdout:\n\nstderr:\ncat:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			stdout, stderr, err := Simple(ctx, tc.args...)

			if diff := cmp.Diff(stdout, tc.wantOut); diff != "" {
				t.Errorf("stdout was not as expected(-got,+want): %s", diff)
			}
			if diff := cmp.Diff(stderr, tc.wantStdErr); diff != "" {
				t.Errorf("stderr was not as expected(-got,+want): %s", diff)
			}
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Error(diff)
			}
		})
	}
}
