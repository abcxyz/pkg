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

// Package mysqltest provides an ephemeral MySQL server for testing database integration. It's
// designed to be used in code that needs to work inside and outside Google.
package mysqltest

import (
	"io"
)

// ConnInfo specifies how to connect to the MySQL server that is created by this package.
type ConnInfo struct {
	Username string
	Password string
	Hostname string
	Port     string
}

// MustStart starts a MySQL server, or panics if there was an error.
func MustStart(opts ...Option) (ConnInfo, io.Closer) {
	conf := buildConfig(opts...)
	ci, closer, err := start(conf)
	if err != nil {
		// The Closer must be called even if there's an error, to clean up the docker container that may
		// exist.
		_ = closer.Close()
		panic(err)
	}

	return ci, closer
}
