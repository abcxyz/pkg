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
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestHandleHTTPHealth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name   string
		header string

		expContentType string
		expBody        string
	}{
		{
			name:           "no_headers",
			header:         "",
			expContentType: genericContentType,
			expBody:        genericResponse,
		},
		{
			name:           "html",
			header:         "text/html; charset=utf-8",
			expContentType: htmlContentType,
			expBody:        htmlResponse,
		},
		{
			name:           "application_json",
			header:         "application/json; charset=utf-8",
			expContentType: jsonContentType,
			expBody:        jsonResponse,
		},
		{
			name:           "text_xml",
			header:         "text/xml; charset=utf-8",
			expContentType: xmlContentType,
			expBody:        xmlResponse,
		},
		{
			name:           "application_xml",
			header:         "application/xml; charset=utf-8",
			expContentType: xmlContentType,
			expBody:        xmlResponse,
		},
		{
			name:           "application_xhtml",
			header:         "application/xhtml+xml; charset=utf-8",
			expContentType: xmlContentType,
			expBody:        xmlResponse,
		},
		{
			name:           "plain_text",
			header:         "text/plain; charset=utf-8",
			expContentType: genericContentType,
			expBody:        genericResponse,
		},
	}

	for _, tc := range cases {
		tc := tc

		for _, header := range []string{"accept", "content-type"} {
			header := header

			t.Run(tc.name+"_"+header, func(t *testing.T) {
				t.Parallel()

				w := httptest.NewRecorder()

				r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}
				r.Header.Set(header, tc.header)

				handler := HandleHTTPHealthCheck()
				handler.ServeHTTP(w, r)

				if got, want := w.Code, http.StatusOK; got != want {
					t.Errorf("expected %d to be %d", got, want)
				}

				if got, want := w.Header().Get("Content-Type"), tc.expContentType; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}

				if got, want := w.Body.String(), tc.expBody; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			})
		}
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
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
