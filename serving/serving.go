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

// Package serving provides an extremely opinionated serving infrastructure,
// with support [net/http.Server] and [google.golang.org/grpc.Server].
//
// It supports listening on specific ports or randomly-available ports, and
// reports the corresponding bind addresses. This is useful for creating
// multiple servers in tests where ports may conflict. The server also
// gracefully stops requests when the provided context is closed.
package serving

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/abcxyz/pkg/logging"
	"google.golang.org/grpc"
)

// Server provides a gracefully-stoppable http server implementation.
type Server struct {
	ip       string
	port     string
	listener net.Listener
}

// New creates a new HTTP server listening on the provided address that responds
// to the http.Handler. It starts the listener, but does not start the server.
// If an empty port is given (or port "0"), the server randomly chooses one.
//
// It binds to all interfaces (0.0.0.0, [::]). To bind to specific interfaces,
// use [NewFromListener] instead.
func New(port string) (*Server, error) {
	// Create the net listener first, so the connection ready when we return. This
	// guarantees that it can accept requests.
	addr := fmt.Sprintf(":" + port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener on %s: %w", addr, err)
	}

	return NewFromListener(listener)
}

// NewFromListener creates a new server on the given listener. This is useful if
// you want to customize the listener type or bind custom networks more than
// [New] allows.
func NewFromListener(listener net.Listener) (*Server, error) {
	netAddr := listener.Addr()
	addr, ok := netAddr.(*net.TCPAddr)
	if !ok {
		return nil, fmt.Errorf("listener is not tcp (got %T)", netAddr)
	}

	return &Server{
		ip:       addr.IP.String(),
		port:     strconv.Itoa(addr.Port),
		listener: listener,
	}, nil
}

// Addr returns the server's listening address (ip + port).
func (s *Server) Addr() string {
	return net.JoinHostPort(s.ip, s.port)
}

// IP returns the server's listening IP.
func (s *Server) IP() string {
	return s.ip
}

// Port returns the server's listening port.
func (s *Server) Port() string {
	return s.port
}

// StartHTTP starts the given [net/http.Server] and blocks until the provided
// context is closed. When the provided context is closed, the HTTP server is
// gracefully stopped with a timeout of 10 seconds; once a server has been
// stopped, it is NOT safe for reuse.
//
// Note that the incoming [net/http.Server]'s address is ignored over the
// listener's configuration.
func (s *Server) StartHTTP(ctx context.Context, srv *http.Server) error {
	logger := logging.FromContext(ctx)

	// Start the server in a background goroutine so we can listen for cancellation
	// in the main process.
	errCh := make(chan error, 1)
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		logger.InfoContext(ctx, "server is starting", "ip", s.ip, "port", s.port)
		defer logger.InfoContext(ctx, "server is stopped")

		if err := srv.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	// Wait for the provided context to finish or an error to occur.
	select {
	case err := <-errCh:
		return fmt.Errorf("failed to serve: %w", err)
	case <-ctx.Done():
		logger.DebugContext(ctx, "provided context is done")
	}

	// Shutdown the server.
	shutdownCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()

	logger.DebugContext(ctx, "server is shutting down")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}
	close(errCh)

	// Wait for the goroutine to finish so we don't leak
	<-doneCh

	return nil
}

// StartHTTPHandler creates and starts a [net/http.Server] with the given
// handler. See [StartHTTP] for more details.
func (s *Server) StartHTTPHandler(ctx context.Context, handler http.Handler) error {
	return s.StartHTTP(ctx, &http.Server{
		// Note: Addr is explicitly ignored because the [Server] has a listener
		// attached.
		Addr: "",

		// Allow custom responses to OPTIONS.
		DisableGeneralOptionsHandler: true,

		// Configure default timeouts.
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,

		// Use the provided handler.
		Handler: handler,
	})
}

// StartGRPC starts the given GRPC service and blocks until the provided context
// is closed. When the provided context is closed, the server is gracefully
// stopped.
//
// Once a server has been stopped, it is NOT safe for reuse.
func (s *Server) StartGRPC(ctx context.Context, srv *grpc.Server) error {
	logger := logging.FromContext(ctx)

	// Start the server in a background goroutine so we can listen for cancellation
	// in the main process.
	errCh := make(chan error, 1)
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		logger.InfoContext(ctx, "server is starting", "ip", s.ip, "port", s.port)
		defer logger.InfoContext(ctx, "server is stopped")

		if err := srv.Serve(s.listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	// Wait for the provided context to finish or an error to occur.
	select {
	case err := <-errCh:
		return fmt.Errorf("failed to serve: %w", err)
	case <-ctx.Done():
		logger.DebugContext(ctx, "provided context is done")
	}

	logger.DebugContext(ctx, "server is shutting down")
	srv.GracefulStop()
	close(errCh)

	// Wait for the goroutine to finish so we don't leak
	<-doneCh

	return nil
}
