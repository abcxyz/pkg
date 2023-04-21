# Mysqltest library

## Introduction

This is a Go library for starting an ephemeral Docker container. It is used to test
integration between Go code and services such as PostgreSQL.

## How to use it

For some containers such as DBs, startup can take a while. You may want to reuse
the same container and `USE` separate databases/namespaces or separate tables to
isolate your tests from each other.

For expensive containers, `Start()` from your `TestMain()` function, If you're not familiar with Go's `TestMain()`
mechanism for global test initialization, see the docs: https://pkg.go.dev/testing#hdr-Main.

Cheaper containers which don't need to be shared can also be started within a test.

MySQL will be used as an example service, though any Service implementation could
be used. Currently, two implementations of `containertest.Service` exist in this
repo, stored in `mysql.go` and `postgres.go`.

```go
package mypackage

import (
    "database/sql"
    "fmt"
    "io"
    "net"
    "os"
    "testing"

    "github.com/abcxyz/pkg/containertest"
    _ "github.com/go-sql-driver/mysql" // Link with the Go MySQL driver
)

var ci *containertest.ConnInfo
var mySQLService *containertest.MySQL

// TestMain runs once at startup to do test initialization common to all tests.
func TestMain(m *testing.M) {
	// Runs unit tests
	os.Exit(func() int {
		mySQLService = &containertest.MySQL{Version: "5.7"}
        var err error // := assignment on next line would shadow ci global variable
		ci, err = containertest.Start(mySQLService) // Start the docker container. Can also pass options.
		if err != nil {
			panic(fmt.Errorf("could not start mysql service: %w", err))
        }
		defer ci.Close()

		return m.Run()
	}())
}

func TestFoo(t *testing.T) {
	t.Parallel()
	// One thing you might want to do is create an SQL driver:

	m := mySQLService
	// Find the port docker exposed for your container
	mySQLPort := ci.PortMapper(m.Port())
	uri := fmt.Sprintf("%s:%s@tcp(%s)/%s", m.Username(), m.Password(),
		net.JoinHostPort(ci.Host, mySQLPort), "")
	db, err := sql.Open("mysql", uri)
	if err != nil {
		t.Fatal(err)
	}

	// application logic goes here
	_ = db
}

```

## A note on leaked Docker containers and timeouts

The `io.Closer` returned from `MustStart()` will terminate the Docker container, which is good, so
it won't sit around wasting resources after the test is done. There is another level of protection
against leaked containers, though: the container will terminate itself after a configurable timeout
(default 10 minutes). If your tests take longer than this, you might need to extend the timeout by
passing `WithTimeout(...)` to `MustStart()`.

## Warning on GitHub Actions

This library doesn't currently work when the Go tests are themselves running inside a Docker
container. It can successfully start a new container, but networking between the two containers
doesn't work. This can cause problems if you configure GitHub Actions in a certain way.

Background: a GitHub Actions workflow runs inside a VM. A workflow consists of multiple steps. Each
step can run in the base VM, or it can run inside a Docker container, for isolation. We recommend running the Go tests inside the VM, not inside a Docker container.

So, in your GitHub actions yaml file, do this:

```yaml
jobs:
  test:
    name: Go Test
    runs-on: ubuntu-latest
    steps:
    - uses: actions/setup-go@v3
      with:
        go-version: '1.20' # Optional
```

... and *don't* do this:

```yaml
jobs:
  test:
    name: Go Test
    runs-on: ubuntu-latest
    container: golang:1.20  # DON'T DO THIS
```
