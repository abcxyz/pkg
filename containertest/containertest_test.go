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
	"fmt"
	"strings"
	"testing"
)

func TestMustStart(t *testing.T) {
	t.Parallel()
	service := (&MySQL{}).WithVersion("8.0")
	ci := MustStart(service, WithLogger(&testLogger{t}))
	defer ci.Close()

	if ci.Host == "" {
		t.Errorf("got empty hostname, wanted a non-empty string")
	}
	if ci.PortMapper(service.Port()) == "" {
		t.Errorf("got empty port, wanted a non-empty string")
	}
}

func TestMustStart_NonexistentVersion(t *testing.T) {
	t.Parallel()

	fakeVersion := "nonexistent_for_test"
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("got no panic, but expected a panic")
		}
		err, ok := r.(error)
		if !ok {
			t.Fatalf("got a %T, but wanted a type that implements the error interface", r)
		}
		wantStr := fmt.Sprintf("tag %q does not exist", fakeVersion)
		if !strings.Contains(err.Error(), wantStr) {
			t.Errorf("got an error %q, but wanted an error containing %q", err.Error(), wantStr)
		}
	}()
	MustStart((&MySQL{}).WithVersion(fakeVersion), WithLogger(&testLogger{t}))
}

func TestBuildConfig(t *testing.T) {
	t.Parallel()

	logger := &testLogger{t}
	conf := buildConfig(
		(&MySQL{}).WithVersion("2"),
		WithKillAfterSeconds(1),
		WithLogger(logger),
	)
	if conf.killAfterSec != 1 {
		t.Errorf("got killAfterSec=%v, want 1", conf.killAfterSec)
	}
	if conf.service.ImageTag() != "2" {
		t.Errorf(`got tag=%v", want "2"`, conf.service.ImageTag())
	}
	if _, ok := conf.progressLogger.(*testLogger); !ok {
		t.Errorf("got progressLogger type %T, want %T", conf.progressLogger, logger)
	}
}

// testLogger is a Logger implementation that passes through to t.Logf. This means that test logs
// are printed for test failures and otherwise hidden, which is convenient.
type testLogger struct {
	tb testing.TB
}

func (tl *testLogger) Printf(fmtStr string, args ...any) {
	tl.tb.Helper()
	tl.tb.Logf(fmtStr, args...)
}
