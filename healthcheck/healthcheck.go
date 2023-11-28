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

// Package healthcheck provides simple health check implementations.
package healthcheck

import (
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	htmlContentType = `text/html; charset=utf-8`
	htmlResponse    = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>Status: ok</title>
  </head>
  <body>
    <p>ok</p>
  </body>
</html>
`

	jsonContentType = `application/json; charset=utf-8`
	jsonResponse    = `{"status":"ok"}`

	xmlContentType = `text/xml; charset=utf-8`
	xmlResponse    = `<?xml version="1.0" encoding="UTF-8"?>
<status>ok</status>
`

	genericContentType = `text/plain`
	genericResponse    = `ok`
)

// HandleHTTPHealthCheck is a basic HTTP health check implementation.
func HandleHTTPHealthCheck() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		val := r.Header.Get("Accept")
		if val == "" {
			val = r.Header.Get("Content-Type")
		}

		switch {
		case strings.HasPrefix(val, "text/html"):
			w.Header().Set("Content-Type", htmlContentType)
			fmt.Fprint(w, htmlResponse)

		case strings.HasPrefix(val, "application/json"):
			w.Header().Set("Content-Type", jsonContentType)
			fmt.Fprint(w, jsonResponse)

		case strings.HasPrefix(val, "text/xml"),
			strings.HasPrefix(val, "application/xml"),
			strings.HasPrefix(val, "application/xhtml+xml"):
			w.Header().Set("Content-Type", xmlContentType)
			fmt.Fprint(w, xmlResponse)

		default:
			w.Header().Set("Content-Type", genericContentType)
			fmt.Fprint(w, genericResponse)
		}
	})
}

// RegisterGRPCHealthCheck registers a basic health check service to the give server.
func RegisterGRPCHealthCheck(grpcServer *grpc.Server) *health.Server {
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, hs)
	return hs
}
