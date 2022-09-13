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

package mysqltest

// This file is only intended to be used outside of Google. Inside of Google, this file should be
// replaced with the Google-internal version.

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestKillAfter(t *testing.T) {
	t.Parallel()

	// This "kill after" time was chosen because it's long enough for the MySQL container to start up.
	// As of 2022-08-31 on MySQL 5.7, it takes 12.5 seconds to start. We add some buffer to leave room
	// for normal variation between test machines.
	const (
		expectedStartupDuration = 13 * time.Second
		extraBuffer             = 10 * time.Second
		killAfter               = expectedStartupDuration + extraBuffer
		killAfterSec            = int(killAfter / time.Second)
	)
	conf := buildConfig(WithKillAfterSeconds(killAfterSec), WithLogger(&testLogger{t}))
	ci, closer, err := start(conf)
	defer func() {
		_ = closer.Close()
	}()
	if err != nil {
		t.Fatal(err)
	}
	db := connect(t, ci)
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}

	deadline := time.Now().Add(killAfter)
	for time.Now().Before(deadline) {
		if err := db.Ping(); err != nil {
			// It would be cleaner to do a type assertion on the error, but the actual type we get is
			// just an *errors.errorString, so we have to examine the text of the error.
			wantOneOf := []string{"bad connection", "invalid connection"}
			if containsOneOf(err.Error(), wantOneOf) {
				t.Log("the docker container stopped itself successfully")
				return
			}
			t.Fatalf("got an error %q, but wanted an error containing one of the substrings %q", err, wantOneOf)
		}
		time.Sleep(200 * time.Millisecond) // Wait a bit between each ping
	}

	t.Errorf("the docker container should have stopped itself by now")
}

// containsOneOf returns whether any of the "needles" are substrings of "haystack".
func containsOneOf(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

func connect(t *testing.T, ci ConnInfo) *sql.DB {
	t.Helper()

	uri := fmt.Sprintf("%s:%s@tcp([%s]:%s)/%s", ci.Username, ci.Password,
		ci.Hostname, ci.Port, "")
	db, err := sql.Open("mysql", uri)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db
}
