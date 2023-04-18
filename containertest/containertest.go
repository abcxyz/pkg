// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package containertest provides an ephemeral container (such as a database) for integration testing.
// It's designed to be used in code that needs to work inside and outside google.
package containertest

import "io"

// ConnInfo specifies how connect to the created container.
type ConnInfo struct {
	Hostname string

	// PortMapper maps from container port to host port. Do not use after container is closed.
	PortMapper func(string) string
}

// Service provides information about what container image should be started and
// how to know when it has finished stating up.
type Service interface {
	ImageRepository() string // Repository for docker image (ex: mysql)
	ImageTag() string        // Tag for docker image (ex: 5.3)
	Environment() []string   // Environment variables to be set in container.
	StartupPorts() []string  // Ports that must be exposed by container before TestConn is run

	// TestConn takes a logger and a mapper to show which ports are exposed, and returns nil if app has started.
	TestConn(progressLogger Logger, portMapper func(string) string) error
}

// MustStart starts a container, or panics if there was an error.
func MustStart(service Service, opts ...Option) (ConnInfo, io.Closer) {
	conf := buildConfig(service, opts...)
	ci, closer, err := start(conf)
	if err != nil {
		// The Closer must be called even if there's an error, to clean up the docker container that may
		// exist.
		_ = closer.Close()
		panic(err)
	}

	return ci, closer
}
