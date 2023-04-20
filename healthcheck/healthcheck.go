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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type HTTPResponse struct {
	Message string
}

// HandleHTTPHealthCheck is a basic HTTP health check implementation.
func HandleHTTPHealthCheck() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		val := r.Header.Get("Accept")
		if val == "" {
			val = r.Header.Get("Content-Type")
		}

		w.Header().Set("Content-Type", val)

		switch {
		case strings.HasPrefix(val, "application/json"):
			if err := json.NewEncoder(w).Encode(&HTTPResponse{Message: "OK"}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		case strings.HasPrefix(val, "application/xml"):
			if err := xml.NewEncoder(w).Encode(&HTTPResponse{Message: "OK"}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		default:
			fmt.Fprint(w, "OK")
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
