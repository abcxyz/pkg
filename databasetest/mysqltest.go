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

// Package databasetest provides an ephemeral database server for testing database integration. It's
// designed to be used in code that needs to work inside and outside Google.
package databasetest

import (
	"database/sql"
	"fmt"
	"io"

	_ "github.com/go-sql-driver/mysql" // Force mysql driver to be included.
)

const (
	// It's OK to hardcode the root password because only boilerplate test data is stored. Also,
	// having a well-known password can help with human inspection for debugging. The value chosen for
	// the password is arbitrary. It can be changed without breaking anything; it's not hardcoded into
	// the docker image or anything like that.
	password = "8mo5lfYKjy6ebTK" //nolint:gosec

	mysqlPort = "3306/tcp"
)

// ConnInfo specifies how to connect to the MySQL server that is created by this package.
type ConnInfo struct {
	Username   string
	Password   string
	Hostname   string
	PortMapper func(string) string
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
	ci.Username = "root"
	ci.Password = password

	return ci, closer
}

func mysqlTester(progressLogger Logger, portMapper func(string) string) error {
	port := portMapper(mysqlPort)
	// Disabling TLS is OK because we're connecting to localhost, and it's just test data.
	addr := fmt.Sprintf("root:%s@tcp(localhost:%s)/mysql?tls=false", password, port)

	progressLogger.Printf(`Checking if MySQL is up yet on localhost at %s. It's normal to see "unexpected EOF" output while it's starting.`, port)
	db, err := sql.Open("mysql", addr)
	if err != nil {
		return fmt.Errorf("sql.Open(): %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("db.Ping(): %w", err)
	}

	progressLogger.Printf("MySQL is up on port %v", port)
	return nil
}
