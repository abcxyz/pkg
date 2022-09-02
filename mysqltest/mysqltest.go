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
	"log"
	"os"
	"testing"
)

var connInfo ConnInfo

// ConnInfo specifies how to connect to the MySQL server that is created by this package.
type ConnInfo struct {
	Username string
	Password string
	Hostname string
	Port     int
}

// Get returns the address and credentials of the MySQL server. The server must already have been
// initialized by calling TestMainHelper(), or this function will panic.
func Get() ConnInfo {
	if connInfo == (ConnInfo{}) {
		// Note to future developers: you might wonder why we don't just initialize the docker container
		// here rather than panic'ing. We decided not to do this because we need to clean up the Docker
		// container when all tests finish. This cleanup logic can only be done from TestMain, so we
		// want to force the callers to create a TestMain.
		panic("mysqltest MySQL server has not been initialized. Call mysqltest.TestMainHelper() from TestMain().")
	}
	return connInfo
}

// TestMainHelper is intended to be run from a TestMain function (see docs at
// https://godoc.corp.google.com/pkg/testing). It handles setting up the Docker container with a
// MySQL server, running the tests, and tearing down the Docker container.
//
// This function calls os.Exit(). It never returns.
//
// The address and credentials of the database can be accessed from test by calling Get().
//
// Example usage:
//
//		func TestMain(m *testing.M) {
//		  mysqltest.TestMainHelper(m)
//		}
//
//		func TestFoo(t *testing.T) {
//	    connInfo := mysqltest.Get()
//		  doSomethingWith(connInfo)
//		  ...
//		}
func TestMainHelper(m *testing.M, opts ...Option) {
	conf := buildConfig(opts...)
	ci, closer, err := start(conf)
	if err != nil {
		// The Closer must be called even if there's an error. We can't use a "defer" because deferred
		// functions don't run in the case of os.Exit() or log.Exit().
		_ = closer.Close()
		log.Fatalf("mysqltest.start(): %v", err)
	}

	connInfo = ci

	out := m.Run()
	if err := closer.Close(); err != nil {
		log.Fatalf("Close(): %v", err)
	}

	os.Exit(out)
}
