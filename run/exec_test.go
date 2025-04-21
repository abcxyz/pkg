package run

import (
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"testing"
)

// this may not be portable
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
