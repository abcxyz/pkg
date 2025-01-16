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
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RegisterFunc is the callback to register a fake gRPC service to the given
// server.
type RegisterFunc func(*grpc.Server)

// FakeGRPCServer creates and registers a fake grpc server for testing. It
// returns the bound address and a connected client. It ensures the server is
// stopped when tests finish. It returns an error if the connection does not
// establish.
func FakeGRPCServer(tb testing.TB, registerFunc RegisterFunc) (string, *grpc.ClientConn) {
	tb.Helper()

	s := grpc.NewServer()
	tb.Cleanup(func() { s.GracefulStop() })

	registerFunc(s)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		tb.Fatalf("net.Listen(tcp, localhost:0) failed: %v", err)
	}

	go func() {
		if err := s.Serve(lis); err != nil {
			tb.Logf("net.Listen(tcp, localhost:0) serve failed: %v", err)
		}
	}()

	addr := lis.Addr().String()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		tb.Fatalf("failed to dail %q: %s", addr, err)
	}
	return addr, conn
}
