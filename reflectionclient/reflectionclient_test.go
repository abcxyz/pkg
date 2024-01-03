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

package reflectionclient

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	api "github.com/abcxyz/pkg/reflectionclient/testing"
)

// FakeServer is a mocked version of a tidal endpoint.
type FakeServer struct {
	resp string
	api.UnimplementedExampleServiceServer
}

// GetConfigVersion implements a function to return a device config version.
func (f *FakeServer) GetConfigVersion(context.Context, *api.GetConfigVersionRequest) (*api.GetConfigVersionResponse, error) {
	return &api.GetConfigVersionResponse{
		Version: &f.resp,
	}, nil
}

// TestClient ensures the reflection client can retrieve info from an unknown grpc service.
func TestClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name     string
		method   string
		payload  string
		resp     string
		wantResp string
		wantErr  string
	}{
		{
			name:     "call_endpoint_without_proper_client",
			method:   "test.ExampleService/GetConfigVersion",
			resp:     "1.0.2",
			wantResp: "{\n  \"version\": \"1.0.2\"\n}\n",
		},
		{
			name:    "remote_method_doesnt_exist",
			method:  "test.ExampleService/GetConfigVersionZ",
			wantErr: `grpcurl.InvokeRPC("test.ExampleService/GetConfigVersionZ") failed: service "test.ExampleService" does not include a method named "GetConfigVersionZ"`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := grpc.NewServer()
			reflection.Register(s)
			defer s.Stop()

			f := FakeServer{resp: tc.resp}
			api.RegisterExampleServiceServer(s, &f)

			lis, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				t.Fatalf("net.Listen(tcp, localhost:0) failed: %v", err)
			}
			go func() {
				err := s.Serve(lis)
				if err != nil {
					t.Errorf("net.Listen(tcp, localhost:0) serve failed: %v", err)
				}
			}()

			client, err := NewClient(
				ctx,
				&ClientConfig{
					Host:     lis.Addr().String(),
					Insecure: true,
					Timeout:  5 * time.Second,
				},
			)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			gotResp, err := client.CallMethod(ctx, tc.method, "", false)
			if err != nil {
				if got, want := err.Error(), tc.wantErr; got != want {
					t.Errorf("ProcessLog() error got=%v, want=%v", got, want)
				}
			} else {
				if got, want := gotResp, tc.wantResp; got.String() != want {
					t.Errorf("ProcessLog() response got=%v, want=%v", got, want)
				}
			}
		})
	}
}
