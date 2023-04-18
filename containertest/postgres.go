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

	_ "github.com/jackc/pgx/v4/stdlib" // Force postgres service to be included.
)

const (
	postgresPort = "5432/tcp"
)

type Postgres struct {
	imageTag string
}

func (p *Postgres) Environment() []string {
	return []string{"POSTGRES_PASSWORD=" + password}
}

func (p *Postgres) ImageRepository() string {
	return "postgres"
}

func (p *Postgres) ImageTag() string {
	return p.imageTag
}
func (p *Postgres) TestConn(progressLogger Logger, portMapper func(string) string) error {
	port := portMapper(postgresPort)
	// Disabling TLS is OK because we're connecting to localhost, and it's just test data.
	addr := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", p.Username(), p.Password(), port, p.Username())

	progressLogger.Printf(`Checking if Postgres is up yet on localhost at %s. It's normal to see "dial error" output while it's starting.`, port)
	db, err := sql.Open("pgx", addr)
	if err != nil {
		return fmt.Errorf("sql.Open(): %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("db.Ping(): %w", err)
	}

	progressLogger.Printf("Postgres is up on port %v", port)
	return nil
}

func (p *Postgres) Port() string {
	return postgresPort
}

func (p *Postgres) StartupPorts() []string {
	return []string{p.Port()}
}

func (p *Postgres) WithVersion(v string) *Postgres {
	p.imageTag = v
	return p
}

func (p *Postgres) Username() string {
	return "postgres"
}
func (p *Postgres) Password() string {
	return password //nolint:gosec
}
