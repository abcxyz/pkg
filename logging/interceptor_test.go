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

package logging

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestGRPCStreamingInterceptor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		headers map[string]string
		exp     string
	}{
		{
			name:    "no_headers",
			headers: nil,
			exp:     "level=INFO msg=test",
		},
		{
			name: "headers_with_no_trace",
			headers: map[string]string{
				"X-Foo": "bar",
			},
			exp: "level=INFO msg=test",
		},
		{
			name: "headers_with_invalid_trace",
			headers: map[string]string{
				"X-Cloud-Trace-Context": "",
			},
			exp: "level=INFO msg=test",
		},
		{
			name: "with_trace_header",
			headers: map[string]string{
				"X-Cloud-Trace-Context": "105445aa7843bc8bf206b12000100000/1;o=1",
			},
			exp: "level=INFO msg=test logging.googleapis.com/trace=projects/my-project/traces/105445aa7843bc8bf206b12000100000",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			originalLogger, buf := testLogger(t)
			ctx := WithLogger(context.Background(), originalLogger)

			streamDesc := &grpc.StreamDesc{
				StreamName: "TestServer.Streamer",
				Handler: func(srv any, stream grpc.ServerStream) error {
					return nil
				},
				ServerStreams: true,
				ClientStreams: true,
			}

			streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
				logger := FromContext(ctx)
				logger.InfoContext(ctx, "test")
				return nil, nil
			}

			// This is a little bit weird - we're setting the _incoming_ context
			// because there's not actually a server here that switches the context
			// from outgoing to incoming.
			var clientConn grpc.ClientConn
			ctx = metadata.NewIncomingContext(ctx, metadata.New(tc.headers))
			interceptor := GRPCStreamingInterceptor(originalLogger, "my-project")
			if _, err := interceptor(ctx, streamDesc, &clientConn, "method", streamer); err != nil {
				t.Fatal(err)
			}

			if got, want := strings.TrimSpace(buf.String()), tc.exp; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}

			if got, want := FromContext(ctx), originalLogger; got != want {
				t.Errorf("expected exact logger on context (%#v vs %#v)", got, want)
			}
		})
	}
}

func TestGRPCUnaryInterceptor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		headers map[string]string
		exp     string
	}{
		{
			name:    "no_headers",
			headers: nil,
			exp:     "level=INFO msg=test",
		},
		{
			name: "headers_with_no_trace",
			headers: map[string]string{
				"X-Foo": "bar",
			},
			exp: "level=INFO msg=test",
		},
		{
			name: "headers_with_invalid_trace",
			headers: map[string]string{
				"X-Cloud-Trace-Context": "",
			},
			exp: "level=INFO msg=test",
		},
		{
			name: "with_trace_header",
			headers: map[string]string{
				"X-Cloud-Trace-Context": "105445aa7843bc8bf206b12000100000/1;o=1",
			},
			exp: "level=INFO msg=test logging.googleapis.com/trace=projects/my-project/traces/105445aa7843bc8bf206b12000100000",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			originalLogger, buf := testLogger(t)
			ctx := WithLogger(context.Background(), originalLogger)

			unaryInfo := &grpc.UnaryServerInfo{
				FullMethod: "TestServer.UnaryInfo",
			}
			unaryHandler := func(ctx context.Context, req any) (any, error) {
				logger := FromContext(ctx)
				logger.InfoContext(ctx, "test")
				return nil, nil
			}

			// This is a little bit weird - we're setting the _incoming_ context
			// because there's not actually a server here that switches the context
			// from outgoing to incoming.
			ctx = metadata.NewIncomingContext(ctx, metadata.New(tc.headers))
			interceptor := GRPCUnaryInterceptor(originalLogger, "my-project")
			if _, err := interceptor(ctx, nil, unaryInfo, unaryHandler); err != nil {
				t.Fatal(err)
			}

			if got, want := strings.TrimSpace(buf.String()), tc.exp; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}

			if got, want := FromContext(ctx), originalLogger; got != want {
				t.Errorf("expected exact logger on context (%#v vs %#v)", got, want)
			}
		})
	}
}

func TestHTTPInterceptor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		headers map[string]string
		exp     string
	}{
		{
			name:    "no_headers",
			headers: nil,
			exp:     "level=INFO msg=test",
		},
		{
			name: "headers_with_no_trace",
			headers: map[string]string{
				"X-Foo": "bar",
			},
			exp: "level=INFO msg=test",
		},
		{
			name: "headers_with_invalid_trace",
			headers: map[string]string{
				"X-Cloud-Trace-Context": "",
			},
			exp: "level=INFO msg=test",
		},
		{
			name: "with_trace_header",
			headers: map[string]string{
				"X-Cloud-Trace-Context": "105445aa7843bc8bf206b12000100000/1;o=1",
			},
			exp: "level=INFO msg=test logging.googleapis.com/trace=projects/my-project/traces/105445aa7843bc8bf206b12000100000",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			originalLogger, buf := testLogger(t)
			ctx := WithLogger(context.Background(), originalLogger)

			r := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}
			r = r.Clone(ctx)

			w := httptest.NewRecorder()

			interceptor := HTTPInterceptor(originalLogger, "my-project")
			interceptor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				logger := FromContext(r.Context())
				logger.InfoContext(ctx, "test")
			})).ServeHTTP(w, r)

			if got, want := strings.TrimSpace(buf.String()), tc.exp; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}

			if got, want := FromContext(ctx), originalLogger; got != want {
				t.Errorf("expected exact logger on context (%#v vs %#v)", got, want)
			}
		})
	}
}

// testLogger creates a logger suitable for testing that writes log messages to
// a buffer. It returns the logger and a pointer to the buffer.
func testLogger(tb testing.TB) (*slog.Logger, *bytes.Buffer) {
	tb.Helper()

	var buf bytes.Buffer
	tb.Cleanup(func() {
		buf.Reset()
	})

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Drop time key for deterministic tests
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	return logger, &buf
}
