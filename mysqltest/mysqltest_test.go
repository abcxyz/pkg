package mysqltest

import (
	"fmt"
	"strings"
	"testing"
)

func TestMustStart(t *testing.T) {
	t.Parallel()

	ci, closer := MustStart(WithLogger(&testLogger{t}))
	defer closer.Close()

	if ci.Username == "" {
		t.Errorf("got empty username, wanted a non-empty string")
	}
	if ci.Password == "" {
		t.Errorf("got empty password, wanted a non-empty string")
	}
	if ci.Hostname == "" {
		t.Errorf("got empty hostname, wanted a non-empty string")
	}
	if ci.Port == "" {
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
		wantStr := fmt.Sprintf("version %q does not exist", fakeVersion)
		if !strings.Contains(err.Error(), wantStr) {
			t.Errorf("got an error %q, but wanted an error containing %q", err.Error(), wantStr)
		}
	}()
	MustStart(WithVersion(fakeVersion), WithLogger(&testLogger{t}))
}

func TestBuildConfig(t *testing.T) {
	logger := &testLogger{t}
	conf := buildConfig(
		WithKillAfterSeconds(1),
		WithVersion("2"),
		WithLogger(logger),
	)
	if conf.killAfterSec != 1 {
		t.Errorf("got killAfterSec=%v, want 1", conf.killAfterSec)
	}
	if conf.mySQLVersion != "2" {
		t.Errorf(`got mySQLVersion=%v", want "2"`, conf.mySQLVersion)
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
