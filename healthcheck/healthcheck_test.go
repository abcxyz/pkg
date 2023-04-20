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

package healthcheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestHandleHTTPHealth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name    string
		code    int
		headers http.Header
		want    string
	}{
		{
			name:    "plain_text_accept_success",
			headers: http.Header{"Accept": []string{"text/plain; charset=utf-8"}},
			code:    200,
			want:    "OK",
		},
		{
			name:    "json_accept_success",
			headers: http.Header{"Accept": []string{"application/json; charset=utf-8"}},
			code:    200,
			want:    `{"Message":"OK"}`,
		},
		{
			name:    "xml_accept_success",
			headers: http.Header{"Accept": []string{"application/xml; charset=utf-8"}},
			code:    200,
			want:    `<HTTPResponse><Message>OK</Message></HTTPResponse>`,
		},
		{
			name:    "plain_text_content_type_success",
			headers: http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
			code:    200,
			want:    "OK",
		},
		{
			name:    "json_content_type_success",
			headers: http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
			code:    200,
			want:    `{"Message":"OK"}`,
		},
		{
			name:    "xml_content_type_success",
			headers: http.Header{"Content-Type": []string{"application/xml; charset=utf-8"}},
			code:    200,
			want:    `<HTTPResponse><Message>OK</Message></HTTPResponse>`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()

			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			for key, values := range tc.headers {
				for _, value := range values {
					r.Header.Set(key, value)
				}
			}

			handler := HandleHTTPHealthCheck()
			handler.ServeHTTP(w, r)

			if got, want := w.Code, tc.code; got != want {
				t.Errorf("expected %d to be %d", got, want)
			}

			if got, want := strings.TrimSpace(w.Body.String()), tc.want; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}

func TestHandleGRPCHealth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	s := grpc.NewServer()

	RegisterGRPCHealthCheck(s)

	t.Cleanup(func() { s.GracefulStop() })

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("net.Listen(tcp, localhost:0) failed: %v", err)
	}

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("net.Listen(tcp, localhost:0) serve failed: %v", err)
		}
	}()

	addr := lis.Addr().String()
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dail %q: %s", addr, err)
	}

	hcClient := healthpb.NewHealthClient(conn)
	res, err := hcClient.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := res.GetStatus(), healthpb.HealthCheckResponse_SERVING; got != want {
		t.Errorf("expected status %v to be %v", got, want)
	}
}
