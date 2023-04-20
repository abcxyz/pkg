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

package containertest

import (
	"database/sql"
	"fmt"
	"net"

	_ "github.com/go-sql-driver/mysql" // Force mysql service to be included.
)

const (
	// It's OK to hardcode the root password because only boilerplate test data is stored. Also,
	// having a well-known password can help with human inspection for debugging. The value chosen for
	// the password is arbitrary. It can be changed without breaking anything; it's not hardcoded into
	// the docker image or anything like that.
	password = "8mo5lfYKjy6ebTK" //nolint:gosec

	mysqlPort = "3306/tcp"
)

// MySQL implements the Service interface, defining a MySQL server container.
type MySQL struct {
	Version string
}

// Environment implements the Service.Environment interface.
func (m *MySQL) Environment() []string {
	return []string{"MYSQL_ROOT_PASSWORD=" + password}
}

// ImageRepository implements the Service.ImageRepository interface.
func (m *MySQL) ImageRepository() string {
	return "mysql"
}

// ImageTag implements the Service.ImageTag interface.
func (m *MySQL) ImageTag() string {
	// Version is the ImageTag that will be returned by the Service interface.
	return m.Version
}

// TestConn implements the Service.TestConn interface.
func (m *MySQL) TestConn(progressLogger TestLogger, connInfo ConnInfo) error {
	port := connInfo.PortMapper(m.Port())
	// Disabling TLS is OK because we're connecting to localhost, and it's just test data.
	addr := fmt.Sprintf("%s:%s@tcp(%s)/mysql?tls=false", m.Username(), m.Password(), net.JoinHostPort(connInfo.Host, port))

	progressLogger.Logf(`Checking if MySQL is up yet on %s. It's normal to see "unexpected EOF" output while it's starting.`, net.JoinHostPort(connInfo.Host, port))
	db, err := sql.Open("mysql", addr)
	if err != nil {
		return fmt.Errorf("sql.Open(): %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("db.Ping(): %w", err)
	}

	progressLogger.Logf("MySQL is up on port %v", port)
	return nil
}

// Port returns the internal port the MySQL container exposes.
func (m *MySQL) Port() string {
	return mysqlPort
}

// StartupPorts implements the Service.StartupPorts interface.
func (m *MySQL) StartupPorts() []string {
	return []string{m.Port()}
}

// Username returns the username for the MySQL database.
func (m *MySQL) Username() string {
	return "root"
}

// Password returns the password for the MySQL database.
func (m *MySQL) Password() string {
	return password
}
