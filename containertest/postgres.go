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

	_ "github.com/jackc/pgx/v4/stdlib" // Force postgres service to be included.
)

const (
	postgresPort = "5432/tcp"
)

// Postgres implements the Service interface, defining a Postgres server container.
type Postgres struct {
	// Version is the ImageTag that will be returned by the Service interface.
	Version string
}

// Environment implements the Service.Environment interface.
func (p *Postgres) Environment() []string {
	return []string{"POSTGRES_PASSWORD=" + password}
}

// ImageRepository implements the Service.ImageRepository interface.
func (p *Postgres) ImageRepository() string {
	return "postgres"
}

// ImageTag implements the Service.ImageTag interface.
func (p *Postgres) ImageTag() string {
	return p.Version
}

// TestConn implements the Service.TestConn interface.
func (p *Postgres) TestConn(progressLogger TestLogger, connInfo *ConnInfo) error {
	port := connInfo.PortMapper(postgresPort)
	// Disabling TLS is OK because we're connecting to localhost, and it's just test data.
	addr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", p.Username(), p.Password(), net.JoinHostPort(connInfo.Host, port), p.Username())

	progressLogger.Logf(`Checking if Postgres is up yet on %s. It's normal to see "dial error" output while it's starting.`, net.JoinHostPort(connInfo.Host, port))
	db, err := sql.Open("pgx", addr)
	if err != nil {
		return fmt.Errorf("sql.Open(): %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("db.Ping(): %w", err)
	}

	progressLogger.Logf("Postgres is up on port %v", port)
	return nil
}

// Port returns the internal port the Postgres container exposes.
func (p *Postgres) Port() string {
	return postgresPort
}

// StartupPorts implements the Service.StartupPorts interface.
func (p *Postgres) StartupPorts() []string {
	return []string{p.Port()}
}

// Username returns the username for the Postgres database.
func (p *Postgres) Username() string {
	return "postgres"
}

// Password returns the password for the Postgres database.
func (p *Postgres) Password() string {
	return password
}
