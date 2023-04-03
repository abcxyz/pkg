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

package serving

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"

	"google.golang.org/grpc"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		port     string
		wantIP   string
		wantPort string
	}{
		{
			name:     "empty_port_uses_random",
			port:     "",
			wantPort: "",
		},
		{
			name:     "port_0_uses_random",
			port:     "0",
			wantPort: "",
		},
		{
			name:     "specific_port",
			port:     "33833", // there's a chance this is taken, but it's low
			wantPort: "33833",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, err := New(tc.port)
			if err != nil {
				t.Fatal(err)
			}

			if got, want := s.port, tc.wantPort; tc.wantPort != "" && got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}

func TestNewFromListener(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		listener func() (net.Listener, error)
		wantAddr string
		wantErr  string
	}{
		{
			name: "tcp_random",
			listener: func() (net.Listener, error) {
				return net.Listen("tcp", ":0")
			},
			wantAddr: "",
		},
		{
			name: "tcp_specific",
			listener: func() (net.Listener, error) {
				return net.Listen("tcp4", ":33834")
			},
			wantAddr: "0.0.0.0:33834",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l, err := tc.listener()
			if err != nil {
				t.Fatal(err)
			}

			s, err := NewFromListener(l)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Fatal(diff)
			}

			if got, want := s.Addr(), tc.wantAddr; tc.wantAddr != "" && got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}

func TestServer_StartHTTP(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	s, err := New("")
	if err != nil {
		t.Fatal(err)
	}

	ctx, done := context.WithCancel(ctx)
	defer done()

	errCh := make(chan error, 1)
	doneCh := make(chan struct{}, 1)
	go func() {
		defer close(doneCh)

		if err := s.StartHTTP(ctx, &http.Server{
			ReadHeaderTimeout: 1 * time.Second,
		}); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	// Stop the server
	done()

	// Read any errors first
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}

	// Wait for done
	select {
	case <-doneCh:
	case <-time.After(500 * time.Millisecond):
		t.Errorf("expected server to be stopped")
	}
}

func TestServer_StartGRPC(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	s, err := New("")
	if err != nil {
		t.Fatal(err)
	}

	ctx, done := context.WithCancel(ctx)
	defer done()

	errCh := make(chan error, 1)
	doneCh := make(chan struct{}, 1)
	go func() {
		defer close(doneCh)

		if err := s.StartGRPC(ctx, grpc.NewServer()); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	// Stop the server
	done()

	// Read any errors first
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}

	// Wait for done
	select {
	case <-doneCh:
	case <-time.After(500 * time.Millisecond):
		t.Errorf("expected server to be stopped")
	}
}
