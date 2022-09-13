package mysqltest

import (
	"fmt"
	"strings"
	"testing"
)

func TestMustStart(t *testing.T) {
	ci, closer := MustStart()
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
	MustStart(WithVersion(fakeVersion))
}

func TestBuildConfig(t *testing.T) {
	logger := &fakeLogger{}
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
	if _, ok := conf.progressLogger.(*fakeLogger); !ok {
		t.Errorf("got progressLogger type %T, want %T", conf.progressLogger, logger)
	}
}

type fakeLogger struct{}

func (*fakeLogger) Printf(string, ...any) {}
