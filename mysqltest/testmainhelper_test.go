// Copyright 2022 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mysqltest_test

// This is file is separate from the rest of the tests because it invokes TestMainHelper, which
// takes over the entire test run.

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/abcxyz/pkg/mysqltest"
)

func TestMain(m *testing.M) {
	mysqltest.TestMainHelper(m)
}

func TestGet(t *testing.T) {
	t.Parallel()

	ci := mysqltest.Get()
	db := connect(t, ci)
	defer db.Close()

	_, err := db.Exec("CREATE DATABASE foo")
	if err != nil {
		t.Fatalf("db.Exec: %v", err)
	}
}

func connect(t *testing.T, ci mysqltest.ConnInfo) *sql.DB {
	t.Helper()

	uri := fmt.Sprintf("%s:%s@tcp([%s]:%d)/%s", ci.Username, ci.Password,
		ci.Hostname, ci.Port, "")
	db, err := sql.Open("mysql", uri)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db
}
