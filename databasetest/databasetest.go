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

// Package databasetest provides an ephemeral database server for integration testing.
// It's designed to be used in code that needs to work inside and outside google.
package databasetest

import "io"

// ConnInfo specifies how to connect to the created container.
type ConnInfo struct {
	Hostname string
	// PortMapper maps from container port to host port. Do not use after container is closed.
	PortMapper func(string) string
}

type Driver interface {
	ImageRepository() string // Repository for docker image (ex: mysql)
	ImageTag() string        // Tag for docker image (ex: 5.3)
	Environment() []string
	StartupPorts() []string                                               // Ports that must be exposed by container before TestConn is run
	TestConn(progressLogger Logger, portMapper func(string) string) error // Function to test if database is up
	// TODO is this needed? WaitForPort
}

// MustStart starts a DB, or panics if there was an error.
func MustStart(driver Driver, opts ...Option) (ConnInfo, io.Closer) {
	conf := buildConfig(driver, opts...)
	ci, closer, err := start(conf)
	if err != nil {
		// The Closer must be called even if there's an error, to clean up the docker container that may
		// exist.
		_ = closer.Close()
		panic(err)
	}

	return ci, closer
}
