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

// Package mysqltest is a legacy compatibility layer for the more generic databasetest
package mysqltest

import (
	"io"

	"github.com/abcxyz/pkg/containertest"
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
	driver := (&containertest.MySQL{}).WithVersion(conf.mySQLVersion)
	realOpts := make([]containertest.Option, 0, 2)
	if conf.killAfterSec != 0 {
		realOpts = append(realOpts, containertest.WithKillAfterSeconds(conf.killAfterSec))
	}
	if conf.progressLogger != nil {
		realOpts = append(realOpts, containertest.WithLogger(conf.progressLogger))
	}

	ci, closer := containertest.MustStart(driver, realOpts...)

	return ConnInfo{
		Username: driver.Username(),
		Password: driver.Password(),
		Hostname: ci.Host,
		Port:     ci.PortMapper(driver.StartupPorts()[0]),
	}, closer
}
