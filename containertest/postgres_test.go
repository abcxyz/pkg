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

package containertest

import (
	"database/sql"
	"fmt"
	"net"
	"testing"
)

func TestPostgres(t *testing.T) {
	t.Parallel()

	p := &Postgres{Version: "15"}
	ci := MustStart(p, WithLogger(t))
	defer ci.Close()

	if ci.Host == "" {
		t.Errorf("got empty hostname, wanted a non-empty string")
	}
	if ci.PortMapper(p.Port()) == "" {
		t.Errorf("got empty port, wanted a non-empty string")
	}

	db := connectPostgres(t, ci, p)

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}
}

func connectPostgres(t *testing.T, ci ConnInfo, p *Postgres) *sql.DB {
	t.Helper()
	addr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", p.Username(), p.Password(), net.JoinHostPort(ci.Host, ci.PortMapper(p.Port())), p.Username())
	db, err := sql.Open("pgx", addr)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db
}
