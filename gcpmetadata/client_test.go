// Copyright 2024 The Authors (see AUTHORS file)
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

package gcpmetadata

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_ProjectID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testServer := testMetadataServer(t)
	client := NewClient(WithHost(testServer.Listener.Addr().String()))

	got, err := client.ProjectID(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := got, "my-project-id"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestClient_ProjectNumber(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testServer := testMetadataServer(t)
	client := NewClient(WithHost(testServer.Listener.Addr().String()))

	got, err := client.ProjectNumber(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := got, "12345"; got != want {
		t.Errorf("expected %q to be %q", got, want)
	}
}

func TestShouldRetry(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		err    error
		exp    bool
	}{
		{
			name:   "200_ok",
			status: 200,
			exp:    false,
		},
		{
			name:   "200_error",
			status: 200,
			err:    fmt.Errorf("oops"),
			exp:    false,
		},
		{
			name:   "500_error",
			status: 500,
			err:    fmt.Errorf("oops"),
			exp:    true,
		},
		{
			name:   "569_ok",
			status: 569,
			exp:    true,
		},
		{
			name:   "eof",
			status: 400,
			err:    fmt.Errorf("oops: %w", io.ErrUnexpectedEOF),
			exp:    true,
		},
		{
			name:   "temporary",
			status: 400,
			err:    &testTemporaryError{},
			exp:    true,
		},
		{
			name:   "unwrappable_error",
			status: 400,
			err:    &testUnwrappableError{},
			exp:    true,
		},
		{
			name:   "unwrappable_errors",
			status: 400,
			err:    &testUnwrappableErrorsError{},
			exp:    true,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got, want := tc.exp, shouldRetry(tc.status, tc.err); got != want {
				t.Errorf("expected retry(%d, %v) %t be %t", tc.status, tc.err, got, want)
			}
		})
	}
}

func testMetadataServer(tb testing.TB) *httptest.Server {
	tb.Helper()

	staticResponse := func(s string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Metadata-Flavor") != "Google" {
				http.Error(w, "missing header", http.StatusBadRequest)
				return
			}

			fmt.Fprint(w, s)
		})
	}

	mux := http.NewServeMux()
	mux.Handle("/computeMetadata/v1/project/project-id", staticResponse("my-project-id"))
	mux.Handle("/computeMetadata/v1/project/numeric-project-id", staticResponse("12345"))

	srv := httptest.NewServer(mux)
	return srv
}

type testTemporaryError struct{}

func (e *testTemporaryError) Error() string {
	return "testTemporaryError"
}

func (e *testTemporaryError) Temporary() bool {
	return true
}

type testUnwrappableError struct{}

func (e *testUnwrappableError) Error() string {
	return "testUnwrappableError"
}

func (e *testUnwrappableError) Unwrap() error {
	return fmt.Errorf("testUnwrappableError: %w", &testTemporaryError{})
}

type testUnwrappableErrorsError struct{}

func (e *testUnwrappableErrorsError) Error() string {
	return "testUnwrappableErrorsError"
}

func (e *testUnwrappableErrorsError) Unwrap() []error {
	return []error{
		nil,
		fmt.Errorf("nope"),
		fmt.Errorf("probably not"),
		fmt.Errorf("testUnwrappableErrorsError: %w", &testTemporaryError{}),
	}
}
