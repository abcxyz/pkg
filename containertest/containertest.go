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

import (
	"fmt"
	"io"
)

// ConnInfo specifies how connect to the created container.
type ConnInfo struct {
	Host string

	// PortMapper maps from container port to host port. Do not use after container is closed.
	PortMapper func(containerPort string) (hostPort string)

	// io.Closer for closing the connection. Should always be initialized, either
	// with an actual closer or io.NoopCloser.
	closer io.Closer
}

// Close implements io.Closer by passing to internal closer field.
func (c ConnInfo) Close() error {
	return fmt.Errorf("error closing container: %w", c.closer.Close())
}

// Service provides information about what container image should be started and
// how to know when it has finished stating up.
type Service interface {
	// ImageRepository returns the repository for docker image (ex: mysql).
	ImageRepository() string

	// ImageTag returns the tag for docker image (ex: 5.3).
	ImageTag() string

	// Environment returns variables to be set in container. Each element is in format of "KEY=VALUE".
	Environment() []string

	// StartupPorts is the list of ports that must be exposed by container before TestConn is run.
	StartupPorts() []string

	// TestConn takes a logger and a struct with connection info, and returns nil if app has started.
	TestConn(progressLogger TestLogger, info *ConnInfo) error
}

// Start starts a container, or returns an error. On err ConnInfo will be automatically
// closed and nil will be returned.
func Start(service Service, opts ...Option) (*ConnInfo, error) {
	conf := buildConfig(service, opts...)
	ci, err := start(conf)
	if err != nil {
		// The Closer must be called even if there's an error, to clean up the docker container that may
		// exist.
		_ = ci.Close()
		return nil, err
	}

	return ci, nil
}
