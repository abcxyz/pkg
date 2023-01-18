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
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	grpcmetadata "google.golang.org/grpc/metadata"
)

var (
	// googleCloudTraceHeader is the header with trace data.
	googleCloudTraceHeader = "X-Cloud-Trace-Context"

	// googleCloudTraceKey is the key in the structured log where trace information
	// is expected to be present.
	googleCloudTraceKey = "logging.googleapis.com/trace"
)

// GRPCStreamingInterceptor returns client-side a gRPC streaming interceptor
// that populates a logger with trace data in the context.
func GRPCStreamingInterceptor(inLogger *slog.Logger, projectID string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Only override the logger if it's the default logger. This is only used
		// for testing and is intentionally a strict object equality check because
		// the default logger is a global default in the logger package.
		logger := inLogger
		if existing := FromContext(ctx); existing != DefaultLogger() {
			logger = existing
		}
		ctx = WithLogger(ctx, logger)

		metadata, ok := grpcmetadata.FromIncomingContext(ctx)
		if ok && len(metadata.Get(googleCloudTraceHeader)) > 0 {
			header := metadata.Get(googleCloudTraceHeader)[0]
			if header != "" {
				ctx = withTracedLogger(ctx, projectID, header)
			}
		}

		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create streaming logging interceptor: %w", err)
		}
		return clientStream, nil
	}
}

// GRPCUnaryInterceptor returns a client-side gRPC unary interceptor that
// populates a logger with trace data in the context.
func GRPCUnaryInterceptor(inLogger *slog.Logger, projectID string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Only override the logger if it's the default logger. This is only used
		// for testing and is intentionally a strict object equality check because
		// the default logger is a global default in the logger package.
		logger := inLogger
		if existing := FromContext(ctx); existing != DefaultLogger() {
			logger = existing
		}
		ctx = WithLogger(ctx, logger)

		metadata, ok := grpcmetadata.FromIncomingContext(ctx)
		if ok && len(metadata.Get(googleCloudTraceHeader)) > 0 {
			header := metadata.Get(googleCloudTraceHeader)[0]
			if header != "" {
				ctx = withTracedLogger(ctx, projectID, header)
			}
		}

		return handler(ctx, req)
	}
}

// HTTPInterceptor returns an HTTP middleware that populates a logger with trace
// data onto the incoming and outgoing [http.Request] context.
func HTTPInterceptor(inLogger *slog.Logger, projectID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Only override the logger if it's the default logger. This is only used
			// for testing and is intentionally a strict object equality check because
			// the default logger is a global default in the logger package.
			logger := inLogger
			if existing := FromContext(ctx); existing != DefaultLogger() {
				logger = existing
			}
			ctx = WithLogger(ctx, logger)

			header := r.Header.Get(googleCloudTraceHeader)
			if header != "" {
				ctx = withTracedLogger(ctx, projectID, header)
			}

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// withTracedLogger is a helper function that extracts the trace information
// from the given header and puts it into the log entry. It is shared among HTTP
// and GRPC interceptors.
func withTracedLogger(ctx context.Context, projectID, header string) context.Context {
	logger := FromContext(ctx)

	// On Google Cloud, extract the trace context and add it to the logger.
	// See: https://cloud.google.com/trace/docs/setup#force-trace
	parts := strings.Split(header, "/")
	if len(parts) > 0 && len(parts[0]) > 0 {
		val := fmt.Sprintf("projects/%s/traces/%s", projectID, parts[0])
		logger = logger.With(googleCloudTraceKey, val)
	}

	return WithLogger(ctx, logger)
}
