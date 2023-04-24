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
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPostgres(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	p := &Postgres{Version: "15"}
	ci, err := Start(p, WithLogger(t))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ci.Host == "" {
		t.Errorf("got empty hostname, wanted a non-empty string")
	}
	if ci.PortMapper(p.Port()) == "" {
		t.Errorf("got empty port, wanted a non-empty string")
	}

	db := connectPostgres(ctx, t, ci, p)

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}
}

func connectPostgres(ctx context.Context, tb testing.TB, ci *ConnInfo, p *Postgres) *pgx.Conn {
	tb.Helper()
	addr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", p.Username(), p.Password(), net.JoinHostPort(ci.Host, ci.PortMapper(p.Port())), p.Username())
	db, err := pgx.Connect(ctx, addr)
	if err != nil {
		tb.Fatalf("pgx.Connect(): %v", err)
	}

	tb.Cleanup(func() {
		_ = db.Close(ctx)
	})

	return db
}
