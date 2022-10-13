# Mysqltest library

## Introduction

This is a Go library for starting an ephemeral MySQL Docker container. It is used to test
integration between Go code and MySQL.

## How to use it

Since it takes about 15 seconds to start the MySQL Docker container, we recommend sharing a single
MySQL instance between all of your tests. `USE` separate databases/namespaces or separate tables to
isolate your tests from each other.

Call `MustStart()` from your `TestMain()` function, If you're not familiar with Go's `TestMain()`
mechanism for global test initialization, see the docs: https://pkg.go.dev/testing#hdr-Main.

```go
package mypackage

import (
    "database/sql"
    "fmt"
    "io"
    "os"
    "testing"

    "github.com/abcxyz/pkg/mysqltest"
    _ "github.com/go-sql-driver/mysql" // Link with the Go MySQL driver
)

var ci mysqltest.ConnInfo

// TestMain runs once at startup to do test initialization common to all tests.
func TestMain(m *testing.M) {
    var closer io.Closer
    ci, closer = mysqltest.MustStart() // Start the docker container. Can also pass options.
    exitCode := m.Run() // Runs unit tests
    
    // Remove container. If tests panic, this won't run, but there's nothing we can do about that:
    // https://github.com/golang/go/issues/37206#issuecomment-590441512. In that case, then the
    // container will eventually time out and be cleaned up.
    closer.Close()

    os.Exit(exitCode)
}

func TestFoo(t *testing.T) {
    // One thing you might want to do is create an SQL driver:
    uri := fmt.Sprintf("%s:%s@tcp([%s]:%s)/", ci.Username, ci.Password, ci.Hostname, ci.Port)
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
container. It can succesfully start a new container, but networking between the two containers
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
        go-version: '1.19' # Optional
```

... and *don't* do this:

```yaml
jobs:
  test:
    name: Go Test
    runs-on: ubuntu-latest
    container: golang:1.19  # DON'T DO THIS
```

