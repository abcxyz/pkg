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

package serving_test

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/abcxyz/pkg/serving"
)

func Example_hTTP() {
	// Any cancellable context will work.
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	httpServer := &http.Server{
		ReadHeaderTimeout: 1 * time.Second,

		Handler: mux,
	}

	server, err := serving.New(os.Getenv("PORT"))
	if err != nil {
		panic(err) // TODO: handle error
	}

	// This will block until the provided context is cancelled.
	if err := server.StartHTTP(ctx, httpServer); err != nil {
		panic(err) // TODO: handle error
	}
}

func Example_hTTPHandler() {
	// Any cancellable context will work.
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	server, err := serving.New(os.Getenv("PORT"))
	if err != nil {
		panic(err) // TODO: handle error
	}

	// This will block until the provided context is cancelled.
	if err := server.StartHTTPHandler(ctx, mux); err != nil {
		panic(err) // TODO: handle error
	}
}

func Example_gRPC() {
	// Any cancellable context will work.
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	grpcServer := grpc.NewServer()

	server, err := serving.New(os.Getenv("PORT"))
	if err != nil {
		panic(err) // TODO: handle error
	}

	// This will block until the provided context is cancelled.
	if err := server.StartGRPC(ctx, grpcServer); err != nil {
		panic(err) // TODO: handle error
	}
}
